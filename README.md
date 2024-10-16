# zitadel-scale-test
scalability of the ZITADEL IAM system by creating a large number of organizations, projects, applications, and users via the ZITADEL API.

# Organization, Project, Application, and User Creation Script
This Go script automates the creation of organizations, projects, applications, and users in a Zitadel environment. It supports both sequential and concurrent execution modes, making it flexible for various use cases.

# Table of Contents
Prerequisites
Usage
Functions Overview
Execution Modes
Logging
Error Handling and Retries

# Prerequisites
Before running the script, ensure you have the following:

Go programming language installed (version 1.16 or higher).

Access to a Zitadel API with valid credentials.

Set up the necessary environment variables or hardcoded values for the API token (apiToken) and base URLs (baseURL, baseURLv2).

# Usage
1.Clone the repository to your local machine:

git clone <repository-url>

cd <repository-directory>

2. Build the Go script:

go build -o app_creation main.go

3. Run the script with the following command:
./app_creation -mode <mode>

Replace <mode> with either sequential or concurrent. The default mode is sequential.

Follow the prompts to enter the number of organizations, projects per organization, applications per project, and users per organization.

# Functions Overview
This script contains the following main functions:

1. createOrganization(orgName string) (string, error)
Creates a new organization with the specified name. If the organization already exists, it retrieves the existing organization's ID.

2. getOrganizationIDByName(orgName string) (string, error)
Fetches the ID of an existing organization by its name.

3. createProject(orgID, projName string) (string, error)
Creates a new project within the specified organization and returns its ID.

4. createApplication(orgID, projID, appName string) (string, error)
Creates a new application within the specified project and organization.

5. createUser(userId, username, givenName, familyName, email, phone, password, orgId string) error
Creates a new human user in the specified organization with the provided details.

6. runSequential(numOrgs, numProjects, numApplications, numUsers int)
Handles the sequential execution of organization, project, application, and user creation.

7. runConcurrent(numOrgs, numProjects, numApplications, numUsers int)
Handles the concurrent execution of organization, project, application, and user creation.

8. retryWithBackoff(attempts int, fn func() error, actionName string) error
Retries a given action with exponential backoff in case of failures.

9. workerPool(workerLimit int, wg *sync.WaitGroup, jobs <-chan func())
Manages a pool of worker goroutines to handle concurrent jobs.

10. generateUniqueName(base string, count int) string
Generates a unique name for organizations, projects, and applications to avoid naming conflicts.

# Execution Modes
The script supports two execution modes:

Sequential Mode: Creates organizations, projects, applications, and users one after the other. This is useful for debugging and understanding the process flow.

./app_creation -mode sequential

Concurrent Mode: Creates organizations, projects, applications, and users simultaneously, which significantly reduces the execution time. This mode is suitable for large-scale operations.

./app_creation -mode concurrent

# Logging
The script logs its operations to an application.log file located in the current directory. It includes detailed information about the success or failure of API requests, as well as timestamps for better traceability.

# Error Handling and Retries
The script implements robust error handling with retry logic for transient failures. If an API request fails, the script will automatically retry up to three times with exponential backoff to manage temporary issues like network interruptions.

