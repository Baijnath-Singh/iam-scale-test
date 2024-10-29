package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

// Structs to represent organization, project, application, and user payload
type Organization struct {
	Name string `json:"name"`
}

type Project struct {
	ID   string `json:"id"` // Assuming the response includes an ID field for the project
	Name string `json:"name"`
}

type Application struct {
	ID   string `json:"id"` // Assuming the response includes an ID field for the application
	Name string `json:"name"`
}

type User struct {
	UserId       string `json:"userId"`
	Username     string `json:"username"`
	Organization struct {
		OrgId string `json:"orgId"`
	} `json:"organization"`
	Profile struct {
		GivenName  string `json:"givenName"`
		FamilyName string `json:"familyName"`
	} `json:"profile"`
	Email struct {
		Email      string `json:"email"`
		IsVerified bool   `json:"isVerified"`
	} `json:"email"`
	Phone struct {
		Phone      string `json:"phone"`
		IsVerified bool   `json:"isVerified"`
	} `json:"phone"`
	Password struct {
		Password       string `json:"password"`
		ChangeRequired bool   `json:"changeRequired"`
	} `json:"password"`
}

var apiToken = "yvx5Z6UMNtZhRlSnYYM_p3bdoXTSajKNEbucqdlM3yTfOB_slZ0fBaawF907xg0JzpN7Hhs" // Replace with your actual API token
var baseURL = "http://127.0.0.1.sslip.io:8080/management/v1"
var baseURLv2 = "http://127.0.0.1.sslip.io:8080/v2" // Replace with actual Zitadel API URL

// Create a single HTTP client to be reused
var client = &http.Client{}

// Function to create organization
func createOrganization(orgName string) (string, error) {
	url := fmt.Sprintf("%s/orgs", baseURL) // Make sure the URL is correct
	method := "POST"

	// Create the payload with the organization name
	payload := map[string]string{"name": orgName}
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %v", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("creating HTTP request for organization: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiToken)

	// Retry logic in case of transient failures (e.g., network issues or 5xx server errors)
	maxRetries := 3
	var resp *http.Response
	for retries := 0; retries < maxRetries; retries++ {
		resp, err = client.Do(req)
		if err != nil {
			if retries < maxRetries-1 {
				log.Printf("Request failed, retrying (%d/%d)...: %v", retries+1, maxRetries, err)
				time.Sleep(2 * time.Second) // Wait before retrying
				continue
			}
			return "", fmt.Errorf("sending request to create organization after %d retries: %v", maxRetries, err)
		}
		break
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %v", err)
	}

	// Log response details for debugging purposes
	log.Printf("Response status: %s", resp.Status)
	log.Printf("Response body: %s", string(body))

	// Handle the "409 Conflict" (organization already exists) case
	if resp.StatusCode == http.StatusConflict {
		log.Printf("Organization %s already exists, proceeding with ID retrieval", orgName)

		// Fetch the organization ID by name since it already exists
		orgID, err := getOrganizationIDByName(orgName)
		if err != nil {
			return "", fmt.Errorf("organization %s exists but failed to fetch ID: %v", orgName, err)
		}

		log.Printf("Fetched existing organization ID: %s", orgID)
		return orgID, nil
	}

	// Check for success (201 Created or 200 OK)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create organization %s, status code: %d, response: %s", orgName, resp.StatusCode, string(body))
	}

	// Parse the response for organization ID
	var orgResponse struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &orgResponse); err != nil {
		return "", fmt.Errorf("decoding organization response: %v", err)
	}

	// Check if the organization ID is returned, otherwise log an error
	if orgResponse.ID == "" {
		return "", fmt.Errorf("organization created but no ID returned in response: %s", string(body))
	}

	log.Printf("Organization created successfully: %s (ID: %s)", orgName, orgResponse.ID)

	return orgResponse.ID, nil
}

// Function to fetch organization ID by its name (assuming an API exists for this)
func getOrganizationIDByName(orgName string) (string, error) {
	url := fmt.Sprintf("%s/orgs/me", baseURL) // Adjust the URL as per Zitadel API to fetch org details
	method := "GET"

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return "", fmt.Errorf("creating HTTP request for fetching org ID: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request to fetch organization: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch organization, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	// Parse the response for organization ID
	var orgResponse struct {
		Org struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"org"`
	}
	if err := json.Unmarshal(body, &orgResponse); err != nil {
		return "", fmt.Errorf("decoding organization response: %v", err)
	}

	// Verify if the organization name matches
	if orgResponse.Org.Name != orgName {
		return "", fmt.Errorf("organization name mismatch, expected: %s, got: %s", orgName, orgResponse.Org.Name)
	}

	return orgResponse.Org.ID, nil
}

// Function to create project
func createProject(orgID, projName string) (string, error) {
	url := fmt.Sprintf("%s/projects", baseURL) // Correct endpoint for project creation
	method := "POST"

	// Create the payload for project creation
	payload := bytes.NewBufferString(fmt.Sprintf(`{
		"name": "%s",
		"projectRoleAssertion": true,
		"projectRoleCheck": true,
		"hasProjectCheck": true,
		"privateLabelingSetting": "PRIVATE_LABELING_SETTING_UNSPECIFIED"
	}`, projName))

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return "", fmt.Errorf("creating HTTP request for project: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiToken)
	req.Header.Add("x-zitadel-orgid", orgID) // Specify the organization ID

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request to create project: %v", err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading project creation response body: %v", err)
	}

	// Log response details for debugging purposes
	log.Printf("Response status: %s", resp.Status)
	log.Printf("Response body: %s", string(body))

	// Treat both 200 OK and 201 Created as valid success cases
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create project %s in organization %s, status code: %d, response: %s", projName, orgID, resp.StatusCode, string(body))
	}

	// Assuming the response body includes the project ID
	var projResponse struct {
		ID string `json:"id"` // Adjust field according to actual response structure
	}
	if err := json.Unmarshal(body, &projResponse); err != nil {
		return "", fmt.Errorf("decoding project response: %v", err)
	}

	if projResponse.ID == "" {
		return "", fmt.Errorf("project created but no ID returned in response: %s", string(body))
	}

	fmt.Printf("Successfully created project: %s in organization: %s\n", projName, orgID)
	return projResponse.ID, nil
}

// Function to create application
func createApplication(orgID, projID, appName string) (string, error) {
	// Construct the URL using the project ID
	url := fmt.Sprintf("%s/projects/%s/apps/api", baseURL, projID)
	method := "POST"

	// Create the payload for application creation
	payload := bytes.NewBufferString(fmt.Sprintf(`{
		"name": "%s",
		"authMethodType": "API_AUTH_METHOD_TYPE_BASIC"
	}`, appName))

	req, err := http.NewRequest(method, url, payload)
	if err != nil {
		return "", fmt.Errorf("creating HTTP request for application: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiToken)
	req.Header.Add("x-zitadel-orgid", orgID) // Specify the organization ID

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending request to create application: %v", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading application creation response body: %v", err)
	}

	// Log response details for debugging purposes
	log.Printf("Response status: %s", resp.Status)
	log.Printf("Response body: %s", string(body))

	// Treat both 200 OK and 201 Created as valid success cases
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to create application %s in project %s, status code: %d, response: %s", appName, projID, resp.StatusCode, string(body))
	}

	// Assuming the response body includes the application details
	var appResponse struct {
		AppId        string `json:"appId"`
		ClientId     string `json:"clientId"`
		ClientSecret string `json:"clientSecret"`
	}
	if err := json.Unmarshal(body, &appResponse); err != nil {
		return "", fmt.Errorf("decoding application response: %v", err)
	}

	// Log application details
	log.Printf("Successfully created application: %s in project: %s", appName, projID)
	log.Printf("App ID: %s, Client ID: %s, Client Secret: %s", appResponse.AppId, appResponse.ClientId, appResponse.ClientSecret)

	return appResponse.AppId, nil
}

// Function to create a human user
func createUser(userId, username, givenName, familyName, email, phone, password, orgId string) error {
	// Construct the URL for creating a new human user
	url := fmt.Sprintf("%s/users/human", baseURLv2) // Replace with your actual domain
	method := "POST"

	// Create the payload for user creation
	userPayload := User{
		UserId:   userId,
		Username: username,
		Organization: struct {
			OrgId string `json:"orgId"`
		}{
			OrgId: orgId,
		},
		Profile: struct {
			GivenName  string `json:"givenName"`
			FamilyName string `json:"familyName"`
		}{
			GivenName:  givenName,
			FamilyName: familyName,
		},
		Email: struct {
			Email      string `json:"email"`
			IsVerified bool   `json:"isVerified"`
		}{
			Email:      email,
			IsVerified: true, // Consider making this configurable
		},
		Phone: struct {
			Phone      string `json:"phone"`
			IsVerified bool   `json:"isVerified"`
		}{
			Phone:      phone,
			IsVerified: true, // Consider making this configurable
		},
		Password: struct {
			Password       string `json:"password"`
			ChangeRequired bool   `json:"changeRequired"`
		}{
			Password:       password,
			ChangeRequired: false,
		},
	}

	// Marshal the user payload to JSON
	payload, err := json.Marshal(userPayload)
	if err != nil {
		return fmt.Errorf("marshalling user payload: %v", err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("creating HTTP request for user: %v", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+apiToken)
	req.Header.Add("x-zitadel-orgid", orgId) // Specify the organization ID

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending request to create user: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Read response body for better error context
		body, _ := ioutil.ReadAll(resp.Body) // Ignore error for simplicity
		return fmt.Errorf("failed to create user %s in organization %s, status code: %d, response: %s", username, orgId, resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully created user: %s\n", username)
	return nil
}

func main() {
	var numOrgs, numProjects, numApplications, numUsers int
	var mode string

	// Initialize logging
	initLogging("application.log")
	log.Println("Application started") // Test log entry

	// Accept the mode of operation as a command-line argument
	flag.StringVar(&mode, "mode", "sequential", "Execution mode: 'sequential' or 'concurrent'")
	flag.Parse()

	// Take inputs from user
	fmt.Print("Enter number of organizations: ")
	_, err := fmt.Scan(&numOrgs)
	if err != nil {
		log.Fatal("Invalid input for number of organizations.")
	}

	fmt.Print("Enter number of projects per organization: ")
	_, err = fmt.Scan(&numProjects)
	if err != nil {
		log.Fatal("Invalid input for number of projects.")
	}

	fmt.Print("Enter number of applications per project: ")
	_, err = fmt.Scan(&numApplications)
	if err != nil {
		log.Fatal("Invalid input for number of applications.")
	}

	fmt.Print("Enter number of users per organization: ")
	_, err = fmt.Scan(&numUsers)
	if err != nil {
		log.Fatal("Invalid input for number of users.")
	}

	// Validate inputs
	if numOrgs <= 0 || numProjects <= 0 || numApplications <= 0 || numUsers <= 0 {
		log.Fatal("All input values must be positive integers.")
	}

	// Check the mode and run accordingly
	switch mode {
	case "concurrent":
		runConcurrent(numOrgs, numProjects, numApplications, numUsers)
	case "sequential":
		runSequential(numOrgs, numProjects, numApplications, numUsers)
	default:
		log.Fatal("Invalid mode. Please choose 'sequential' or 'concurrent'.")
	}
}

func runSequential(numOrgs, numProjects, numApplications, numUsers int) {
	fmt.Println("Running in sequential mode...")

	// Initialize counters for created entities
	var orgCount, projectCount, appCount, userCount int

	// Create organizations, projects, applications, and users (sequentially)
	for i := 0; i < numOrgs; i++ {
		orgName := fmt.Sprintf("org-%d", i+1)

		// Create organization
		orgId, err := createOrganization(orgName)
		if err != nil {
			log.Fatalf("Error creating organization %s: %v", orgName, err)
		}
		orgCount++

		// Create projects for each organization
		for j := 0; j < numProjects; j++ {
			projName := fmt.Sprintf("project-%d", j+1)
			projId, err := createProject(orgId, projName)
			if err != nil {
				log.Fatalf("Error creating project %s: %v", projName, err)
			}
			projectCount++

			// Create applications for each project
			for k := 0; k < numApplications; k++ {
				appName := fmt.Sprintf("app-%d", k+1)
				_, err := createApplication(orgId, projId, appName)
				if err != nil {
					log.Fatalf("Error creating application %s: %v", appName, err)
				}
				appCount++
			}
		}

		// Create users for each organization
		for l := 0; l < numUsers; l++ {
			userId := fmt.Sprintf("user-%d-org-%d", l+1, i+1)
			userName := fmt.Sprintf("user-%d-org-%d", l+1, i+1)
			givenName := fmt.Sprintf("GivenName%d", l+1)
			familyName := fmt.Sprintf("FamilyName%d", l+1)
			email := fmt.Sprintf("user%d-org%d@example.com", l+1, i+1)
			phone := fmt.Sprintf("+123456789%d", l)
			password := "Secret@1234"

			err := createUser(userId, userName, givenName, familyName, email, phone, password, orgId)
			if err != nil {
				log.Fatalf("Error creating user %s: %v", userName, err)
			}
			userCount++
		}
	}

	// Print summary
	fmt.Printf("\nTotal Organizations Created: %d\n", orgCount)
	fmt.Printf("Total Projects Created: %d\n", projectCount)
	fmt.Printf("Total Applications Created: %d\n", appCount)
	fmt.Printf("Total Users Created: %d\n", userCount)
}

// Constants for retry logic
const maxRetries = 3
const initialBackoff = time.Millisecond * 100

// Worker pool size to limit concurrent goroutines
const workerPoolSize = 100

// Create exponential backoff with time tracking
func retryWithBackoff(attempts int, fn func() error, actionName string) error {
	backoff := initialBackoff
	for i := 0; i < attempts; i++ {
		start := time.Now() // Start the timer for the API call
		err := fn()
		duration := time.Since(start) // Calculate the duration of the API call

		if err == nil {
			log.Printf("%s succeeded. Time taken: %v\n", actionName, duration)
			return nil
		}
		log.Printf("%s failed on attempt %d after %v. Retrying...\n", actionName, i+1, duration)
		if i < attempts-1 {
			time.Sleep(backoff)
			backoff *= 2
		}
	}
	return fmt.Errorf("after %d attempts, last error: %v", attempts, fn())
}

// Worker pool to control concurrency
func workerPool(workerLimit int, wg *sync.WaitGroup, jobs <-chan func()) {
	sem := make(chan struct{}, workerLimit)
	for job := range jobs {
		sem <- struct{}{} // Acquire worker
		wg.Add(1)
		go func(job func()) {
			defer wg.Done()
			defer func() { <-sem }() // Release worker
			job()
		}(job)
	}
}

func generateUniqueName(base string, count int) string {
	// Random number generation for uniqueness
	randomSuffix := rand.Intn(10000)                          // Random number between 0 and 9999
	return fmt.Sprintf("%s-%d-%d", base, count, randomSuffix) // Add random number for uniqueness
}

func runConcurrent(numOrgs, numProjects, numApplications, numUsers int) {
	fmt.Println("Running in concurrent mode...")

	startTotal := time.Now() // Start tracking total execution time

	var wg sync.WaitGroup
	var mu sync.Mutex

	// Initialize counters for created entities
	var orgCount, projectCount, appCount, userCount int

	// Create channels for jobs
	orgJobs := make(chan func(), numOrgs)
	projectJobs := make(chan func(), numProjects*numOrgs)
	appJobs := make(chan func(), numApplications*numProjects*numOrgs)
	userJobs := make(chan func(), numUsers*numOrgs)

	// Create a worker pool to handle org, project, app, and user creation concurrently
	go workerPool(workerPoolSize, &wg, orgJobs)
	go workerPool(workerPoolSize, &wg, projectJobs)
	go workerPool(workerPoolSize, &wg, appJobs)
	go workerPool(workerPoolSize, &wg, userJobs)

	// Create organizations concurrently
	for i := 0; i < numOrgs; i++ {
		orgName := generateUniqueName("org", i+1) // Unique org name
		wg.Add(1)                                 // Add to WaitGroup before submitting the job
		orgJobs <- func() {
			defer wg.Done() // Mark job as done when finished
			err := retryWithBackoff(maxRetries, func() error {
				orgId, err := createOrganization(orgName)
				if err != nil {
					return err
				}

				// Create projects, apps, and users for each organization
				for j := 0; j < numProjects; j++ {
					projName := generateUniqueName(orgName+"-project", j+1) // Unique project name
					wg.Add(1)                                               // Add to WaitGroup before submitting the project job
					projectJobs <- func() {
						defer wg.Done() // Mark job as done when finished
						err := retryWithBackoff(maxRetries, func() error {
							projId, err := createProject(orgId, projName)
							if err != nil {
								return err
							}

							mu.Lock()
							projectCount++
							mu.Unlock()

							// Create applications for the project
							for k := 0; k < numApplications; k++ {
								appName := generateUniqueName(projName+"-app", k+1) // Unique application name
								wg.Add(1)                                           // Add to WaitGroup before submitting the application job
								appJobs <- func() {
									defer wg.Done() // Mark job as done when finished
									err := retryWithBackoff(maxRetries, func() error {
										_, err := createApplication(orgId, projId, appName)
										if err != nil {
											return err
										}

										mu.Lock()
										appCount++
										mu.Unlock()
										return nil
									}, fmt.Sprintf("Create Application: %s", appName))
									if err != nil {
										log.Printf("Error creating application %s: %v", appName, err)
									}
								}
							}
							return nil
						}, fmt.Sprintf("Create Project: %s", projName))
						if err != nil {
							log.Printf("Error creating project %s: %v", projName, err)
						}
					}
				}

				// Create users for the organization
				for l := 0; l < numUsers; l++ {
					userName := generateUniqueName(orgName+"-user", l+1) // Unique user name
					wg.Add(1)                                            // Add to WaitGroup before submitting the user job
					userJobs <- func() {
						defer wg.Done() // Mark job as done when finished
						err := retryWithBackoff(maxRetries, func() error {
							userId := fmt.Sprintf("user-%d-org-%s", l+1, orgId)
							givenName := fmt.Sprintf("GivenName%d", l+1)
							familyName := fmt.Sprintf("FamilyName%d", l+1)
							email := fmt.Sprintf("user%d-org%s@example.com", l+1, orgId)
							phone := fmt.Sprintf("+123456789%d", l)
							password := "Secret@1234"

							err := createUser(userId, userName, givenName, familyName, email, phone, password, orgId)
							if err != nil {
								return err
							}

							mu.Lock()
							userCount++
							mu.Unlock()
							return nil
						}, fmt.Sprintf("Create User: %s", userName))
						if err != nil {
							log.Printf("Error creating user %s: %v", userName, err)
						}
					}
				}

				mu.Lock()
				orgCount++
				mu.Unlock()
				return nil
			}, fmt.Sprintf("Create Organization: %s", orgName))
			if err != nil {
				log.Printf("Error creating organization %s: %v", orgName, err)
			}
		}
	}

	// Close job channels only after all jobs have been submitted and processed
	go func() {
		wg.Wait() // Wait for all jobs to be added and processed
		close(orgJobs)
		close(projectJobs)
		close(appJobs)
		close(userJobs)
	}()

	// Wait for all goroutines to finish
	wg.Wait()

	// Track total execution time
	totalDuration := time.Since(startTotal)

	// Print summary
	fmt.Printf("\nTotal Organizations Created: %d\n", orgCount)
	fmt.Printf("Total Projects Created: %d\n", projectCount)
	fmt.Printf("Total Applications Created: %d\n", appCount)
	fmt.Printf("Total Users Created: %d\n", userCount)
	fmt.Printf("Total Time Taken: %v\n", totalDuration)
}

func initLogging(logFilePath string) {
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening log file: %v", err)
	}
	log.SetOutput(file)

	// Add this line to confirm logging is initialized
	log.Println("Logging initialized successfully")
}
