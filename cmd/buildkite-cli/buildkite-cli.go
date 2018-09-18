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

func get_token() string {

	home_path := os.Getenv("HOME")

	cfg, err := ini.Load(fmt.Sprintf("%s/.buildkite/config", home_path))
	if err != nil {
		log.Fatalln(err)
	}

	token := cfg.Section("default").Key("token").String()

	if len(token) == 0 {
		log.Fatalln("buildkite token is empty")
	}

	return token
}

func http_request(token string, url string, method string) string {

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	token_header := fmt.Sprintf("Bearer %s", token)
	req.Header.Set("Authorization", token_header)
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

func get_pipelines(token string, api_endpoint string, organization string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines", api_endpoint, organization)
	return http_request(token, url, "GET")
}

func get_log(token string, api_endpoint string, organization string, pipeline string, build_number string, job_id string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s/jobs/%s/log", api_endpoint, organization, pipeline, build_number, job_id)
	return http_request(token, url, "GET")
}

func get_latest_build(token string, api_endpoint string, organization string, pipeline string) string {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds", api_endpoint, organization, pipeline)
	response := http_request(token, url, "GET")
	return gjson.Get(response, "0.number").String()
}

func get_job_ids(token string, api_endpoint string, organization string, pipeline string, build string, follow bool) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s", api_endpoint, organization, pipeline, build)
	response := http_request(token, url, "GET")

	ids := gjson.Get(response, "jobs.#.id")
	for _, id := range ids.Array() {
		fmt.Println(get_log(token, api_endpoint, organization, pipeline, build, id.String()))
	}
}

func find_commit(token string, api_endpoint string, organization string, pipeline string, commit string, follow bool) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds?commit=%s", api_endpoint, organization, pipeline, commit)
	response := http_request(token, url, "GET")
	build := gjson.Get(response, "0.number").String()
	get_job_ids(token, api_endpoint, organization, pipeline, build, follow)
}

func trigger_build(token string, api_endpoint string, organization string, pipeline string, build string) {
	url := fmt.Sprintf("%s/organizations/%s/pipelines/%s/builds/%s/rebuild", api_endpoint, organization, pipeline, build)
	http_request(token, url, "PUT")
}

func list_agents(token string, api_endpoint string, organization string) {
	url := fmt.Sprintf("%s/organizations/%s/agents", api_endpoint, organization)
	response := http_request(token, url, "GET")
	fmt.Println(response)
}

var (
	api_endpoint = kingpin.Flag("api-endpoint", "Buildkite API endpoint").Default("https://api.buildkite.com/v2").String()
	organization = kingpin.Flag("organization", "Buildkite organization").Default("stellar").String()
	pipeline     = kingpin.Flag("pipeline", "Buildkite Pipeline").Short('p').String()
	build        = kingpin.Flag("build", "Buildkite bulild number").Short('b').String()
	commit       = kingpin.Flag("commit", "Commit hash").Short('c').String()
	trigger      = kingpin.Flag("trigger", "Trigger a build").Short('t').Bool()
	follow       = kingpin.Flag("follow", "follow build output").Short('f').Bool()
	agents       = kingpin.Flag("agents", "list agents").Short('a').Bool()
	version      = kingpin.Flag("-version", "list agents").Short('v').Bool()
)

var Version = "No Version Produced"

func main() {

	kingpin.UsageTemplate(kingpin.CompactUsageTemplate).Version("1.0").Author("Denis Khoshaba")
	kingpin.CommandLine.Help = "Buildkite cli tool"
	kingpin.Parse()

	token := get_token()

	if *version {
		fmt.Printf("Version: %s\n", Version)
		os.Exit(0)
	}

	if *agents {
		list_agents(token, *api_endpoint, *organization)
		os.Exit(0)
	}

	if *trigger {
		if len(*pipeline) > 0 && len(*build) > 0 {
			trigger_build(token, *api_endpoint, *organization, *pipeline, *build)
		} else if len(*pipeline) > 0 && len(*build) == 0 {
			build := get_latest_build(token, *api_endpoint, *organization, *pipeline)
			trigger_build(token, *api_endpoint, *organization, *pipeline, build)
		}
		os.Exit(0)
	}

	if len(*pipeline) > 0 && len(*build) == 0 && len(*commit) == 0 {
		build := get_latest_build(token, *api_endpoint, *organization, *pipeline)
		get_job_ids(token, *api_endpoint, *organization, *pipeline, build, *follow)
	} else if len(*pipeline) > 0 && len(*build) > 0 && len(*commit) == 0 {
		get_job_ids(token, *api_endpoint, *organization, *pipeline, *build, *follow)
	} else if len(*pipeline) > 0 && len(*build) == 0 && len(*commit) > 0 {
		find_commit(token, *api_endpoint, *organization, *pipeline, *commit, *follow)
	}
	os.Exit(0)
}
