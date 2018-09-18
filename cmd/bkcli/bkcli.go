package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/tidwall/gjson"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/ini.v1"
)

func getToken() string {

	homePath := os.Getenv("HOME")

	cfg, err := ini.Load(fmt.Sprintf("%s/.bkcli/config", homePath))
	if err != nil {
		log.Fatalln(err)
	}

	token := cfg.Section("default").Key("token").String()

	if len(token) == 0 {
		log.Fatalln("buildkite token is empty")
	}

	return token
}

func httprequest(token string, url string, method string) string {

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	tokenHeader := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", tokenHeader)
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

func getPipelines(token string, apiEndpoint string, organization string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines", apiEndpoint, organization)
	return httprequest(token, url, "GET")
}

func getLog(token string, apiEndpoint string, organization string, pipeline string, build string, jobID string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s/jobs/%s/log", apiEndpoint, organization, pipeline, build, jobID)
	return httprequest(token, url, "GET")
}

func getLatestBuild(token string, apiEndpoint string, organization string, pipeline string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds", apiEndpoint, organization, pipeline)
	response := httprequest(token, url, "GET")
	return gjson.Get(response, "0.number").String()
}

func getJobIds(token string, apiEndpoint string, organization string, pipeline string, build string, follow bool) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s", apiEndpoint, organization, pipeline, build)
	response := httprequest(token, url, "GET")

	ids := gjson.Get(response, "jobs.#.id")
	for _, id := range ids.Array() {
		fmt.Println(getLog(token, apiEndpoint, organization, pipeline, build, id.String()))
	}
}

func findCommit(token string, apiEndpoint string, organization string, pipeline string, commit string, follow bool) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds?commit=%s", apiEndpoint, organization, pipeline, commit)
	response := httprequest(token, url, "GET")
	build := gjson.Get(response, "0.number").String()
	getJobIds(token, apiEndpoint, organization, pipeline, build, follow)
}

func triggerBuild(token string, apiEndpoint string, organization string, pipeline string, build string) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s/rebuild", apiEndpoint, organization, pipeline, build)
	httprequest(token, url, "PUT")
}

func listAgents(token string, apiEndpoint string, organization string) {
	url := fmt.Sprintf("%s/organizations/%s/agents", apiEndpoint, organization)
	response := httprequest(token, url, "GET")
	fmt.Println(response)
}

var (
	apiEndpoint  = kingpin.Flag("api-endpoint", "Buildkite API endpoint").Default("https://api.buildkite.com/v2").String()
	organization = kingpin.Flag("organization", "Buildkite organization").Default("stellar").String()
	pipeline     = kingpin.Flag("pipeline", "Buildkite Pipeline").Short('p').String()
	build        = kingpin.Flag("build", "Buildkite bulild number").Short('b').String()
	commit       = kingpin.Flag("commit", "Commit hash").Short('c').String()
	trigger      = kingpin.Flag("trigger", "Trigger a build").Short('t').Bool()
	follow       = kingpin.Flag("follow", "follow build output").Short('f').Bool()
	agents       = kingpin.Flag("agents", "list agents").Short('a').Bool()
)

// Version number to be passed in during build
var Version = "No Version Produced"

func main() {

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("1.0").Author("Denis Khoshaba")
	kingpin.CommandLine.Help = "Buildkite cli tool"
	kingpin.Parse()

	token := getToken()

	if *agents {
		listAgents(token, *apiEndpoint, *organization)
		os.Exit(0)
	}

	if *trigger {
		if len(*pipeline) > 0 && len(*build) > 0 {
			triggerBuild(token, *apiEndpoint, *organization, *pipeline, *build)
		} else if len(*pipeline) > 0 && len(*build) == 0 {
			build := getLatestBuild(token, *apiEndpoint, *organization, *pipeline)
			triggerBuild(token, *apiEndpoint, *organization, *pipeline, build)
		}
		os.Exit(0)
	}

	if len(*pipeline) > 0 && len(*build) == 0 && len(*commit) == 0 {
		build := getLatestBuild(token, *apiEndpoint, *organization, *pipeline)
		getJobIds(token, *apiEndpoint, *organization, *pipeline, build, *follow)
	} else if len(*pipeline) > 0 && len(*build) > 0 && len(*commit) == 0 {
		getJobIds(token, *apiEndpoint, *organization, *pipeline, *build, *follow)
	} else if len(*pipeline) > 0 && len(*build) == 0 && len(*commit) > 0 {
		findCommit(token, *apiEndpoint, *organization, *pipeline, *commit, *follow)
	}
	os.Exit(0)
}
