package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk" // Update the import path
)

// Configurable settings
var (
	numOrgs            int          // Total number of organizations to create
	numGoroutines      int          // Number of goroutines for parallel creation
	organizationPrefix = "TestOrg_" // Prefix for unique organization names
	logFile            *os.File     // File to log output
)

// Struct to hold timing data
type TimingInfo struct {
	orgName  string
	duration time.Duration
}

// Initialize Casdoor SDK configuration
func initializeCasdoor() {
	casdoorsdk.InitConfig("http://localhost:8000", "785e6f4416906c6d3598", "d8b784ac15ca6acd9fc372810679d62e57e81ecc", "your_jwt_public_key", "built-in", "app-built-in")
}

// Function to create an organization with unique name
func createOrganization(orgID int, wg *sync.WaitGroup, timings chan<- TimingInfo) {
	defer wg.Done()

	// Generate unique name
	orgName := fmt.Sprintf("%s%d_%d", organizationPrefix, orgID, rand.Intn(10000))

	// Create organization struct
	organization := &casdoorsdk.Organization{
		Owner:              "admin",
		Name:               orgName,
		CreatedTime:        time.Now().Format("2006-01-02T15:04:05Z"),
		DisplayName:        orgName,
		WebsiteUrl:         "https://example.com",
		PasswordType:       "plain",
		PasswordOptions:    []string{"AtLeast6"},
		CountryCodes:       []string{"US"},
		Languages:          []string{"en"},
		InitScore:          1000,
		EnableSoftDeletion: false,
		IsProfilePublic:    false,
	}

	// Measure time taken for creation
	startTime := time.Now()
	success, err := casdoorsdk.AddOrganization(organization)
	duration := time.Since(startTime)

	// Log the result and send timing info to the channel
	if err != nil || !success {
		log.Printf("Failed to create organization %s: %v\n", orgName, err)
	} else {
		log.Printf("Successfully created organization %s in %v\n", orgName, duration)
	}

	timings <- TimingInfo{orgName: orgName, duration: duration}
}

// Setup logging to a file
func setupLogging() {
	var err error
	logFile, err = os.OpenFile("org_creation.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Error opening log file: %v", err)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile) // Set log flags
}

func main() {
	// Seed the random number generator
	rand.Seed(time.Now().UnixNano())

	// Setup logging to a file
	setupLogging()
	defer logFile.Close() // Ensure the log file is closed when the program exits

	// Initialize the SDK
	initializeCasdoor()

	// User inputs
	fmt.Print("Enter number of organizations to create: ")
	fmt.Scan(&numOrgs)
	fmt.Print("Enter number of goroutines (parallelism): ")
	fmt.Scan(&numGoroutines)

	// Channels and wait group for concurrency and timing
	timings := make(chan TimingInfo, numOrgs)
	var wg sync.WaitGroup

	// Create goroutines in batches
	for i := 0; i < numOrgs; i += numGoroutines {
		for j := 0; j < numGoroutines && (i+j) < numOrgs; j++ {
			wg.Add(1)
			go createOrganization(i+j, &wg, timings)
		}
		wg.Wait() // Wait for the batch to complete before moving to next
	}

	// Collect timing results
	close(timings)
	var totalDuration time.Duration
	createdOrgs := 0

	for timing := range timings {
		totalDuration += timing.duration
		createdOrgs++
	}

	// Calculate average time
	avgDuration := totalDuration / time.Duration(createdOrgs)
	fmt.Printf("Total organizations created: %d\n", createdOrgs)
	fmt.Printf("Average time taken per organization: %v\n", avgDuration)

	// Log final results
	log.Printf("Total organizations created: %d\n", createdOrgs)
	log.Printf("Average time taken per organization: %v\n", avgDuration)
}
