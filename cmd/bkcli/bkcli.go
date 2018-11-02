package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tidwall/gjson"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/ini.v1"
)

// Grabs organization from ~/.bkcli/config (ini format)
func getOrg(profile string) string {
	homePath := os.Getenv("HOME")

	cfg, err := ini.Load(fmt.Sprintf("%s/.bkcli/config", homePath))
	if err != nil {
		log.Fatalln(err)
	}

	organization := cfg.Section(profile).Key("organization").String()

	if len(organization) == 0 {
		log.Fatalln("buildkite org given is empty")
	}

	return organization

}

// Grabs token from ~/.bkcli/config (ini format)
func getToken(profile string) string {

	homePath := os.Getenv("HOME")

	cfg, err := ini.Load(fmt.Sprintf("%s/.bkcli/config", homePath))
	if err != nil {
		log.Fatalln(err)
	}

	token := cfg.Section(profile).Key("token").String()

	if len(token) == 0 {
		log.Fatalln("buildkite token is empty")
	}

	return token
}

// Wrapper function for requests to the API
func httprequest(token string, url string, method string) string {

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	tokenHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", tokenHeader)
	// Use text/plain to ensure log output is in ansi
	req.Header.Set("Accept", "text/plain")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	body := buf.String()

	return body

}

// Returns the list of pipelines
func getPipelines(token string, apiEndpoint string, organization string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines", apiEndpoint, organization)
	return httprequest(token, url, "GET")
}

// Returns the log output for a job
func getLog(token string, apiEndpoint string, organization string, pipeline string, build string, jobID string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s/jobs/%s/log", apiEndpoint, organization, pipeline, build, jobID)
	return httprequest(token, url, "GET")
}

// Returns the latest build for a pipeline
func getLatestBuild(token string, apiEndpoint string, organization string, pipeline string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds", apiEndpoint, organization, pipeline)
	response := httprequest(token, url, "GET")
	return gjson.Get(response, "0.number").String()
}

func showFailedSteps(token string, apiEndpoint string, organization string, pipeline string, build string) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s", apiEndpoint, organization, pipeline, build)
	response := httprequest(token, url, "GET")
	states := gjson.Get(response, "jobs.#.state").Array()
	for index, state := range states {
		if state.String() == "failed" {
			fmt.Println(gjson.Get(response, fmt.Sprintf("jobs.%v.name", index)))
		}
	}
}

func getJobIds(token string, apiEndpoint string, organization string, pipeline string, build string, follow bool, pollRate time.Duration) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s", apiEndpoint, organization, pipeline, build)
	response := httprequest(token, url, "GET")

	// Get a list of all job ids
	ids := gjson.Get(response, "jobs.#.id")

	// If the follow flag is pased attemp to do a "tail -f"
	if follow {
		jobStatus := gjson.Get(response, "jobs.#.finished_at")
		jobCount := len(jobStatus.Array())
		for index, id := range jobStatus.Array() {
			// Ensure jobs array is less than the jobs with statuses
			if index < jobCount {
				log := getLog(token, apiEndpoint, organization, pipeline, build, ids.Array()[index].String())
				// Count the lines of the log
				lines := strings.Count(log, "\n")
				// If "finished_at" key is nonempty the job is finished, so we just print the full log
				if len(id.String()) > 0 {
					if lines > 0 {
						fmt.Println(log)
					}
					// Otherwise if the finished_at" key is empty, job isn't complete to "tail -f" is attempted
				} else {
					fmt.Println(log)
					oldlines := 0
					// Loop until the jobStatus ("finished_at") is nonempty
					for len(jobStatus.Array()[index].String()) <= 0 {
						if lines > 0 {
							split := strings.Split(log, "\n")
							// If lines have changed between "polls", print the new lines
							if oldlines != lines {
								for i := oldlines; i < len(split); i++ {
									fmt.Println(split[i])
								}
							}
							jobStatus = gjson.Get(httprequest(token, url, "GET"), "jobs.#.finished_at")
							log = getLog(token, apiEndpoint, organization, pipeline, build, ids.Array()[index].String())
							oldlines = lines
							lines = strings.Count(log, "\n")
							// Check for new log output at pollRate
							time.Sleep(pollRate)
							// break polling if no more lines are left
						} else {
							break
						}
					}
				}
			}
		}
	} else {
		// If no follow flag is passed, loop through all job Ids and print the logs
		for _, id := range ids.Array() {
			log := getLog(token, apiEndpoint, organization, pipeline, build, id.String())
			fmt.Println(log)
		}
	}
}

// Returns the build id for a commit hash
func findCommit(token string, apiEndpoint string, organization string, pipeline string, commit string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds?commit=%s", apiEndpoint, organization, pipeline, commit)
	response := httprequest(token, url, "GET")
	build := gjson.Get(response, "0.number").String()
	return build
}

// Triggers a build
func triggerBuild(token string, apiEndpoint string, organization string, pipeline string, build string) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s/rebuild", apiEndpoint, organization, pipeline, build)
	httprequest(token, url, "PUT")
}

// Returns the list of buildkite Agents
func listAgents(token string, apiEndpoint string, organization string) {
	url := fmt.Sprintf("%s/organizations/%s/agents", apiEndpoint, organization)
	response := httprequest(token, url, "GET")
	fmt.Println(response)
}

var (
	apiEndpoint  = kingpin.Flag("api-endpoint", "Buildkite API endpoint").Default("https://api.buildkite.com/v2").String()
	organization = kingpin.Flag("organization", "Buildkite organization").String()
	pipeline     = kingpin.Flag("pipeline", "Buildkite Pipeline").Short('p').String()
	build        = kingpin.Flag("build", "Buildkite bulild number").Short('b').String()
	commit       = kingpin.Flag("commit", "Commit hash").Short('c').String()
	trigger      = kingpin.Flag("trigger", "Trigger a build").Short('t').Bool()
	follow       = kingpin.Flag("follow", "Follow build output").Short('f').Bool()
	agents       = kingpin.Flag("agents", "List agents").Short('a').Bool()
	profile      = kingpin.Flag("profile", "token profile").Default("default").String()
	pollRate     = kingpin.Flag("pollrate", "Rate at which to poll api when following logs").Default("2s").Duration()
	showfailed   = kingpin.Flag("show-failed", "Show the failed step(s) for a build").Bool()
)

// Version number to be passed in during compile time
var Version = "non-versioned build"

func main() {

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version(Version).Author("Denis Khoshaba")
	kingpin.CommandLine.Help = "Buildkite cli tool"
	kingpin.Parse()

	// Print help if no arguments are passed
	if len(os.Args) == 1 {
		kingpin.Usage()
		os.Exit(0)
	}

	// Environment variable for the token takes precedence
	token := os.Getenv("BUILDKITE_TOKEN")
	if len(token) == 0 {
		token = getToken(*profile)
	}

	// Flag for org takes precedence over the env var then profile
	if len(*organization) == 0 {
		*organization = os.Getenv("BUILDKITE_ORG")
		if len(*organization) == 0 {
			*organization = getOrg(*profile)
		}
	}

	// Check if agents flag is passed
	if *agents {
		listAgents(token, *apiEndpoint, *organization)
		os.Exit(0)
	}

	// First check if user wants to trigger a build before log checking
	if *trigger {
		if len(*pipeline) > 0 && len(*build) > 0 {
			triggerBuild(token, *apiEndpoint, *organization, *pipeline, *build)
		} else if len(*pipeline) > 0 && len(*build) == 0 {
			build := getLatestBuild(token, *apiEndpoint, *organization, *pipeline)
			triggerBuild(token, *apiEndpoint, *organization, *pipeline, build)
		}
		os.Exit(0)
	}

	// Log checking for different cases
	// First check if just a pipeline name is passed
	if len(*pipeline) > 0 && len(*build) == 0 && len(*commit) == 0 {
		build := getLatestBuild(token, *apiEndpoint, *organization, *pipeline)
		getJobIds(token, *apiEndpoint, *organization, *pipeline, build, *follow, *pollRate)
		os.Exit(0)
	}

	// If a pipeline name and build number args are passed
	if len(*pipeline) > 0 && len(*build) > 0 && len(*commit) == 0 {
		if *showfailed {
			showFailedSteps(token, *apiEndpoint, *organization, *pipeline, *build)
		} else {
			getJobIds(token, *apiEndpoint, *organization, *pipeline, *build, *follow, *pollRate)
			os.Exit(0)
		}
	}

	// If a pipeline name and commit hash args are passed
	if len(*pipeline) > 0 && len(*build) == 0 && len(*commit) > 0 {
		build := findCommit(token, *apiEndpoint, *organization, *pipeline, *commit)
		getJobIds(token, *apiEndpoint, *organization, *pipeline, build, *follow, *pollRate)
		os.Exit(0)
	}
}
