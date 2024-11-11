package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

type Event struct {
    Type string `json:"type"`
    Repo struct {
        Name string `json:"name"`
    } `json:"repo"`
    Payload struct {
        Action  string `json:"action"`
        Commits []struct{} `json:"commits"`
    } `json:"payload"`
}

type Response struct {
    Activities []Activity `json:"activities"`
}

type Activity struct {
    Type     string `json:"type"`
    Message  string `json:"message"`
    RepoName string `json:"repo_name"`
}

func fetchGithubActivity(username string, eventType string) ([]Activity, error) {
    url := fmt.Sprintf("https://api.github.com/users/%s/events", username)
    client := &http.Client{}
    req, _ := http.NewRequest("GET", url, nil)
    
    // Add GitHub token if available
    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        req.Header.Add("Authorization", "token "+token)
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to fetch data: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
        return nil, fmt.Errorf("API returned status code %d", resp.StatusCode)
    }

    var events []Event
    if err := json.NewDecoder(resp.Body).Decode(&events); err != nil {
        return nil, fmt.Errorf("failed to parse JSON: %v", err)
    }

    activities := []Activity{}
    for _, event := range events {
        if eventType != "" && event.Type != eventType {
            continue
        }
        
        activity := Activity{
            Type:     event.Type,
            RepoName: event.Repo.Name,
        }

        switch event.Type {
        case "PushEvent":
            activity.Message = fmt.Sprintf("Pushed %d commits", len(event.Payload.Commits))
        case "IssuesEvent":
            activity.Message = fmt.Sprintf("%s an issue", event.Payload.Action)
        case "WatchEvent":
            activity.Message = "Starred repository"
        default:
            activity.Message = event.Type
        }
        
        activities = append(activities, activity)
    }

    return activities, nil
}

func handleActivity(w http.ResponseWriter, r *http.Request) {
    username := r.URL.Query().Get("username")
    eventType := r.URL.Query().Get("type")

    if username == "" {
        http.Error(w, "username parameter is required", http.StatusBadRequest)
        return
    }

    activities, err := fetchGithubActivity(username, eventType)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(Response{Activities: activities})
}

func main() {
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    http.HandleFunc("/api/activity", handleActivity)
    log.Printf("Server starting on port %s", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}