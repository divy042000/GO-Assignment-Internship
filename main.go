package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "log"
    "os"
    "strconv" // Add this line to import the strconv package
     "sync"
    // "github.com/sirupsen/logrus"
    "go.uber.org/zap"
)

// User represents a user in the messaging application
type User struct {
    ID       string
    Messages []string
}
// CatFactResponse is the structure of the JSON response from the Cat Facts API
type CatFactResponse struct {
    Fact string `json:"fact"`
    Length int    `json:"length"`
}
type Message struct {
    SenderID    string
    RecipientID string
    Content     string
}

type Channel struct {
    ID      string
    Messages []Message
   }
// AppState represents the state of the messaging application
type AppState struct {
    Users map[string]*User
    CentralChannel Channel
    Logger *zap.Logger
    
}

var (
	mu       sync.Mutex
	userLogs map[string][]string
)
func init() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	userLogs = make(map[string][]string)
}
func logMessage(userID, message string) {
	mu.Lock()
	defer mu.Unlock()

	userLogs[userID] = append(userLogs[userID], message)
}
func displayUserLogs(userID string) {
	mu.Lock()
	defer mu.Unlock()

	if logs, ok := userLogs[userID]; ok {
		for _, log := range logs {
			fmt.Printf("[%s]: %s\n", userID, log)
		}
	} else {
		fmt.Printf("No logs found for user %s\n", userID)
	}
}
func displayAllLogs() {
	mu.Lock()
	defer mu.Unlock()

	for userID, logs := range userLogs {
		fmt.Printf("User %s logs:\n", userID)
		for _, log := range logs {
			fmt.Printf("[%s]: %s\n", userID, log)
		}
	}
}

func handleUserInput(appState *AppState) {
    for {
        fmt.Println("Enter your choice:")
        fmt.Println("1. Add a user")
        fmt.Println("2. Send a message")
        fmt.Println("3. View messages")
        fmt.Println("4. Broadcast a message")
        fmt.Println("5. Exit")

        var choice int
        _, err := fmt.Scan(&choice)
        if err != nil {
            log.Println(err)
            continue
        }

        switch choice {
        case 1:
            handleAddUser(appState)
        case 2:
            handleSendMessage(appState)
        case 3:
            handleViewMessages(appState)
        case 4:
            handleBroadcastMessage(appState)
        case 5:
            handleExit(appState)
            return
        default:
            fmt.Println("Invalid choice. Please try again.")
        }
    }
}


func handleAddUser(appState *AppState) {
    fmt.Println("Enter the number of users you want to add:")
    var numUsers int
    _, err := fmt.Scan(&numUsers)
    if err != nil {
        log.Println(err)
        return
    }

    for i := 0; i < numUsers; i++ {
        fmt.Printf("Enter user ID %d (integer):\n", i+1)
        var userID int
        _, err := fmt.Scan(&userID)
        if err != nil {
            log.Println(err)
            return
        }

        // Convert the integer userID to a string to use as a key in the map
        userIDStr := strconv.Itoa(userID)

        if _, exists := appState.Users[userIDStr]; exists {
            fmt.Printf("User ID %d already exists. Please choose a different user ID.\n", userID)
            continue // Skip to the next iteration of the loop
        }

        appState.Users[userIDStr] = &User{
            ID:       userIDStr, // Store the user ID as a string
            Messages: []string{},
        }
        fmt.Printf("User ID %d added successfully.\n", userID)
    }
}



func handleSendMessage(appState *AppState) {
	fmt.Println("Enter the sender's ID:")
	var senderID string
	_, err := fmt.Scan(&senderID)
	if err != nil {
		appState.Logger.Error("Error reading sender ID", zap.Error(err))
		return
	}

	fmt.Println("Enter the recipient's ID:")
	var recipientID string
	_, err = fmt.Scan(&recipientID)
	if err != nil {
		appState.Logger.Error("Error reading recipient ID", zap.Error(err))
		return
	}

	fmt.Println("Enter your message:")
	var message string
	_, err = fmt.Scan(&message)
	if err != nil {
		appState.Logger.Error("Error reading message", zap.Error(err))
		return
	}

	if message == "" {
		message = getRandomFact()
	}

	// Check if both sender and recipient are in the Users map
	senderExists := false
	recipientExists := false

	// Check if the sender ID exists in the Users map
	if _, exists := appState.Users[senderID]; exists {
		senderExists = true
	}

	// Check if the recipient ID exists in the Users map
	if _, exists := appState.Users[recipientID]; exists {
		recipientExists = true
	}

	if !senderExists {
		appState.Logger.Warn("Sender ID does not exist", zap.String("SenderID", senderID))
		return
	}

	if !recipientExists {
		appState.Logger.Warn("Recipient ID does not exist", zap.String("RecipientID", recipientID))
		return
	}

	// Send the message to the recipient using the central channel
	appState.Logger.Info("Message sent",
		zap.String("sender", senderID),
		zap.String("recipient", recipientID),
		zap.String("message", message),
	)

	// Append the message to the central channel
	appState.CentralChannel.Messages = append(appState.CentralChannel.Messages, Message{
		SenderID:    senderID,
		RecipientID: recipientID,
		Content:     message,
	})

	// Log the message for the sender and recipient
	logMessage(senderID, message)
	logMessage(recipientID, message)

	// Print the message details
	fmt.Printf("Message from %s to %s: %s\n", senderID, recipientID, message)
}

func handleViewMessages(appState *AppState) {
    fmt.Println("Enter your user ID to view your messages:")
    var userID string
    _, err := fmt.Scan(&userID)
    if err != nil {
        appState.Logger.Error("Error reading user ID", zap.Error(err))
        return
    }

    // Check if the user exists in the Users map
    if _, exists := appState.Users[userID]; !exists {
        appState.Logger.Warn("User ID does not exist", zap.String("UserID", userID))
        fmt.Printf("No user found with ID %s\n", userID)
        return
    }

    // Iterate through the CentralChannel.Messages to find messages for the user
    fmt.Printf("Messages for user %s:\n", userID)
    for _, message := range appState.CentralChannel.Messages {
        if message.SenderID == userID || message.RecipientID == userID ||message.RecipientID == "BROADCAST" {
            fmt.Printf("From %s to %s: %s\n", message.SenderID, message.RecipientID, message.Content)
        }
    }
}

func handleBroadcastMessage(appState *AppState) {
    fmt.Println("Enter your message:")
    var message string
    _, err := fmt.Scan(&message)
    if err != nil {
        appState.Logger.Error("Error reading message", zap.Error(err))
        return
    }

    if message == "" {
        message = getRandomFact() // Assuming getRandomFact() is a function that returns a random fact
    }

    // Assuming the sender is the user who initiated the broadcast, we use the environment variable "USER" to get the sender's ID
    senderID := os.Getenv("USER")
    if sender, exists := appState.Users[senderID]; exists {
        // Log the broadcast message using zap's logging methods
        appState.Logger.Info("Message broadcasted",
            zap.String("sender", sender.ID),
            zap.String("message", message),
        )

        // Create a new message struct for the broadcast message
        broadcastMessage := Message{
            SenderID:    sender.ID,
            RecipientID: "BROADCAST", // Indicating this is a broadcast message
            Content:     message,
        }

        // Append the broadcast message to the CentralChannel.Messages
        appState.CentralChannel.Messages = append(appState.CentralChannel.Messages, broadcastMessage)

        fmt.Println("Broadcast message sent successfully.")
    } else {
        appState.Logger.Warn("Sender ID does not exist", zap.String("SenderID", senderID))
        fmt.Printf("No user found with ID %s\n", senderID)
    }
}

// handleExit handles the application exit, displaying all central messages
func handleExit(appState *AppState) {
    fmt.Println("Exiting the application. Here are the central message logs:")
    for _, message := range appState.CentralChannel.Messages {
        fmt.Printf("From %s to %s: %s\n", message.SenderID, message.RecipientID, message.Content)
    }
}

// getRandomFact fetches a random fact from the provided API
func getRandomFact() string {
    // URL of the Cat Facts API
    url := "https://catfact.ninja/fact"

    // Make the HTTP request
    resp, err := http.Get(url)
    if err != nil {
        fmt.Println("Error fetching random cat fact:", err)
        return "Error fetching random cat fact."
    }
    defer resp.Body.Close()

    // Read the response body
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println("Error reading response body:", err)
        return "Error reading response body."
    }

    // Parse the JSON response
    var catFactResponse CatFactResponse
    err = json.Unmarshal(body, &catFactResponse)
    if err != nil {
        fmt.Println("Error parsing JSON:", err)
        return "Error parsing JSON."
    }

    // Return the random cat fact
    return catFactResponse.Fact
}

func main() {
    // Initialize the application state
    appState := &AppState{
        Users: make(map[string]*User),
    }

    // Initialize the logger
    logger, err := zap.NewProduction()
    if err != nil {
        // Handle the error, for example, by logging it and exiting the program
        log.Fatalf("Failed to initialize logger: %v", err)
    }
    appState.Logger = logger

    // Initialize the user data
    appState.Users[os.Getenv("USER")] = &User{ID: os.Getenv("USER"), Messages: []string{}}

    // Start the application
    handleUserInput(appState)
}
