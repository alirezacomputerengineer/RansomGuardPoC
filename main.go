package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"log/syslog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/rivo/tview"
)

const encryptionKey = "hardcoded-encryption-key" // Simple hardcoded key
const configFile = "config.cfg"

// Config holds the configuration details for ransomguard
type Config struct {
	iftttURL      string         `json:"IFTTT_URL"`
	HoneypotFiles []HoneypotFile `json:"honeypot_files"`
}

// HoneypotFile represents the configuration for a honeypot file.
type HoneypotFile struct {
	Name      string `json:"name"`
	Extension string `json:"extensions"`
	Volume    int    `json:"volume"` // Size in kilobytes
	Route     string `json:"route"`
}

// Alert represents an alert generated by the ransomguard.
type Alert struct {
	Description string
	ProcessName string
	ProcessID   int
}

func main() {
	//Show Logo and Welcome Message and Slogan
	asciiArt := `
 *****************************************************************************************
 *****************************************************************************************
 ** 	 _____                                  _____                           _       **
 **	|  __ \                                / ____|                         | |      **
 **	| |__) |__ _ _ __  ___  ___  _ __ ___ | |  __  ___  _   _  __ _ _ __ __| |      **
 **	|  _  // _' | '_ \/ __|/ _ \| '_ ' _ \| | |_ |/ _ \| | | |/ _' | '__/ _' |      **
 **	| | \ \ (_| | | | \__ \ (_) | | | | | | |__| | (_) | |_| | (_| | | | (_| |      **
 **	|_|  \_\__,_|_| |_|___/\___/|_| |_| |_|\_____|\___/ \__,_|\__,_|_|  \__,_|      **
 **                                                                                     **
 **                                                                                     **
 *****************************************************************************************
 *****************************************************************************************
`
	welcomeMessage := "**                                 RansomGuard Ver 1.0                                 **"
	slogan := "**                   The Ultimate Defense Against Ransomware Attacks                   **"
	separator := "*****************************************************************************************"
	fmt.Println(asciiArt, welcomeMessage, "\n", separator, "\n", slogan, "\n", separator, "\n", separator)

	// Load configuration from the encrypted config file
	config, err := internal.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Initialize honeypot files if they don't already exist
	err = internal.CreateHoneypotFiles(config)
	if err != nil {
		log.Fatalf("Error initializing honeypot files: %v", err)
	}

	// Channel for OS signals to allow graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Channel to signal an issue, such as ransomware detection, across all goroutines
	alertChan := make(chan internal.Alert)

	// Start monitoring honeypot files for any modifications
	go func() {
		err := internal.MonitorHoneypotFiles(config, alertChan)
		if err != nil {
			log.Fatalf("Error in honeypot file watching: %v", err)
		}
	}()
	// Monitor for alerts from all goroutines
	go func() {
		for alert := range alertChan {
			handleAlert(alert, config)
		}
	}()

	// Wait for termination signals to gracefully exit
	<-sigChan
	log.Println("Ransomguard is shutting down.")
}

func handleAlert(alertChan <-chan Alert, config Config) {

	log.Printf("ALERT: %s detected on process: %s", alert.Description, alert.ProcessName)
	// Open a connection to syslog
	logWriter, err := syslog.New(syslog.LOG_ALERT|syslog.LOG_USER, "RansomGuard")
	if err != nil {
		fmt.Printf("Error connecting to syslog: %v\n", err)
		return
	}
	defer logWriter.Close()

	// Initialize the TUI application
	app := tview.NewApplication()

	// Listen for alerts on the channel
	for alert := range alertChan {
		// Step 1: Log the alert to syslog
		syslogMsg := fmt.Sprintf("RANSOMWARE_DETECTED: %s - Process: %s (PID: %d)", alert.Description, alert.ProcessName, alert.ProcessID)
		logWriter.Alert(syslogMsg)
		fmt.Println("Alert sent to syslog:", syslogMsg) // for local confirmation

		// Step 2: Send IFTTT Alert
		err = sendIFTTTAlert(alert, config)
		if err != nil {
			log.Printf("Error sending alert to IFTTT: %v", err)
		}
		// Create and display the TUI pop-up alert
		modal := tview.NewModal().
			SetText(fmt.Sprintf("⚠️ WARNING: %s\nProcess: %s (PID: %d)", alert.Description, alert.ProcessName, alert.ProcessID)).
			AddButtons([]string{"Acknowledge"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				app.Stop() // Close the app when acknowledged
			})
		// Set the modal as the root and run the application
		if err := app.SetRoot(modal, true).Run(); err != nil {
			fmt.Printf("Error displaying TUI alert: %v\n", err)
		}
		// Step 3: Stop the process that triggered the alert
		err := TerminateProcess(alert.ProcessID)
		if err != nil {
			log.Printf("Failed to terminate process %d: %v", alert.ProcessID, err)
		} else {
			log.Printf("Process %d terminated successfully.", alert.ProcessID)
		}

		// Step 4: Send the process file to quarantine
		err = QuarantineProcess(alert.ProcessName)
		if err != nil {
			log.Printf("Failed to quarantine process: %v", err)
		} else {
			log.Printf("Process %s moved to quarantine.", alert.ProcessName)
		}
	}
}

func sendIFTTTAlert(alert Alert, config Config) error {
	jsonData, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %v", err)
	}

	req, err := http.NewRequest("POST", config.iftttURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send alert to IFTTT: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("IFTTT returned non-200 status: %v", resp.Status)
	}
	fmt.Println("Alert sent to IFTTT:", alert)
	return nil
}

func CreateHoneypotFiles(config *Config) error {
	for _, honeypot := range config.HoneypotFiles {
		filePath := filepath.Join(honeypot.Route, honeypot.Name+honeypot.Extension)

		// Check if file already exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// Create a new honeypot file
			err := createFileWithVolume(filePath, honeypot.Volume)
			if err != nil {
				log.Printf("Failed to create honeypot file %s: %v", filePath, err)
				return err
			}
			log.Printf("Honeypot file created: %s", filePath)
		} else {
			log.Printf("Honeypot file already exists: %s", filePath)
		}
	}
	return nil
}

// createFileWithVolume creates a file and fills it with dummy data to reach the specified volume in KB
func createFileWithVolume(filePath string, volumeKB int) error {
	// Create the directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return fmt.Errorf("error creating directory: %v", err)
	}

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Fill the file with dummy data to match the volume in kilobytes
	data := make([]byte, 1024) // 1 KB of dummy data
	for i := 0; i < volumeKB; i++ {
		_, err := file.Write(data)
		if err != nil {
			return fmt.Errorf("error writing to file: %v", err)
		}
	}

	return nil
}

// MonitorHoneypotFile monitors honeypot files for modifications and sends alerts
func MonitorHoneypotFiles(config *Config, alertChan chan Alert) {
	for _, honeypot := range config.HoneypotFiles {
		filePath := filepath.Join(honeypot.Route, honeypot.Name+honeypot.Extension)
		go monitorFile(filePath, alertChan)
	}
}

func monitorFile(h HoneypotFile, alertChan chan Alert) {
	// Create a new watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating file watcher: %v", err)
	}
	defer watcher.Close()

	// Add the honeypot file to the watcher
	err = watcher.Add(h.Route)
	if err != nil {
		log.Fatalf("Error adding file to watcher: %v", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Check if the event is a write (modification)
			if event.Op&fsnotify.Write == fsnotify.Write {
				processName, processID, err := getProcessDetails(h.Route)
				alert := Alert{
					Description: fmt.Sprintf("Modified: %s - %s", event.Name, h.Name),
					ProcessName: processName,
					ProcessID:   processID,
				}
				alertChan <- alert
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Error watching file: %v", err)
		}
	}
}

// LoadConfig reads the config file, decrypts it, and unmarshals the JSON data
func LoadConfig() (*Config, error) {
	encryptedData, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Println("Config file not found.")
		return nil, errors.New("config file not found")
	}

	// Decrypt data
	decryptedData, err := DecryptData(encryptedData, encryptionKey)
	if err != nil {
		log.Println("Failed to decrypt config file.")
		return nil, errors.New("failed to decrypt config file")
	}

	// Unmarshal JSON
	var config Config
	err = json.Unmarshal(decryptedData, &config)
	if err != nil {
		log.Println("Failed to unmarshal config file.")
		return nil, errors.New("failed to parse config file")
	}

	log.Println("Configuration loaded successfully.")
	return &config, nil
}

func EncryptData(data []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	encryptedData := gcm.Seal(nonce, nonce, data, nil)
	return []byte(base64.StdEncoding.EncodeToString(encryptedData)), nil
}

// DecryptData decrypts the given encrypted data using the hardcoded key
func DecryptData(encryptedData []byte, key string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(string(encryptedData))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// getProcessDetails retrieves the name and PID of the process that modified the file
func getProcessDetails(filePath string) (string, int, error) {
	// Iterate over each PID directory in /proc
	procDirs, err := os.ReadDir("/proc")
	if err != nil {
		return "", 0, fmt.Errorf("failed to read /proc: %v", err)
	}

	for _, procDir := range procDirs {
		// Only look at directories with numeric names (these represent process IDs)
		if pid, err := strconv.Atoi(procDir.Name()); err == nil {
			fdDir := filepath.Join("/proc", procDir.Name(), "fd")
			fdFiles, err := os.ReadDir(fdDir)
			if err != nil {
				continue // skip if we can't access the fd directory
			}

			for _, fdFile := range fdFiles {
				fdPath := filepath.Join(fdDir, fdFile.Name())
				linkPath, err := os.Readlink(fdPath)
				if err != nil {
					continue // skip if we can't read the symlink
				}

				if linkPath == targetPath {
					// Get the process name
					commPath := filepath.Join("/proc", procDir.Name(), "comm")
					processName, err := os.ReadFile(commPath)
					if err != nil {
						return "", pid, fmt.Errorf("failed to read process name: %v", err)
					}
					return string(processName), pid, nil
				}
			}
		}
	}
	return "", 0, fmt.Errorf("no process found accessing %s", targetPath)
}

// TerminateProcess terminates the given process by process ID (PID).
func TerminateProcess(pid int) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("failed to find process with PID %d: %v", pid, err)
	}

	err = process.Signal(syscall.SIGTERM)
	if err != nil {
		return fmt.Errorf("failed to terminate process with PID %d: %v", pid, err)
	}

	// Wait for a moment to allow graceful termination
	time.Sleep(2 * time.Second)
	if err = process.Signal(syscall.SIGKILL); err != nil {
		return fmt.Errorf("failed to force kill process with PID %d: %v", pid, err)
	}

	return nil
}

// QuarantineProcess moves the executable file associated with the process to a quarantine directory.
func QuarantineProcess(processName string) error {
	quarantineDir := "/var/quarantine"
	err := os.MkdirAll(quarantineDir, 0750)
	if err != nil {
		return fmt.Errorf("failed to create quarantine directory: %v", err)
	}

	processPath, err := exec.LookPath(processName)
	if err != nil {
		return fmt.Errorf("failed to find the executable path for process: %s, %v", processName, err)
	}

	quarantinePath := filepath.Join(quarantineDir, filepath.Base(processPath))

	err = os.Rename(processPath, quarantinePath)
	if err != nil {
		return fmt.Errorf("failed to move process to quarantine: %v", err)
	}

	return nil
}
