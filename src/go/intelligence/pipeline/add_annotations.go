package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/intelligence/models"
	"go/intelligence/utils"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

// Lock variable for solving Concurrent Read/Write on Map issue.
var lock sync.RWMutex

//AddAnnotation - function for adding annotations to dashboards
func AddAnnotation(data <-chan models.AmJSON) <-chan models.AmJSON {

	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("AddAnnotation step")
	}

	dataAfterAnnotation := make(chan models.AmJSON)

	go func() {
		defer close(dataAfterAnnotation)
		ptr := &utils.DCache
		ptr.UpdateDashboardCache()

		for each := range data {
			if utils.ConfigJSON.Server.Verbose > 1 {
				log.Println(each.String())
			}
			var srv models.Service
			ifServiceFound := false

			for _, service := range utils.ConfigJSON.Services {
				lock.RLock()
				slabel, ok := each.Labels[utils.ConfigJSON.Alerts.ServiceLabel]
				lock.RUnlock()
				if ok && slabel == service.Name {
					srv = service
					ifServiceFound = true
					break
				}
			}

			if ifServiceFound {

				for _, annotationData := range srv.AnnotationMap.AnnotationsData {

					ifActionFound := checkIfAvailable(annotationData.Actions, each, srv.AnnotationMap.Label) //If action keywords match in alerts
					ifSystemFound := checkIfAvailable(annotationData.Systems, each, srv.AnnotationMap.Label) //If system keywords match in alerts

					if ifActionFound && ifSystemFound {

						for _, dashboard := range utils.DCache.Dashboards {

							/*
								ifTagsIntersect(annotationDashboardTags, allDashboardsTags []string) bool {..} is used for checking if there's intersection between two list of strings each having tags for Grafana Dashboards.

								Check is done by finding the length of intersection result.

								If it is greater than 0 --> Means there are some tags after intersection.
								Else --> there are no tags after intersection

								Each service has annotationMap field, and annotationMap has a "annotation" field which is a list of action & system keywords and tags for dashboards.
								See below in the example.

								"annotationMap": {
										.
										.
										.

										"annotations": [
											{
												"actions": ["intervention", "outage"],
												"systems": ["network", "database", "db"],	---> #
												"tags": ["cmsweb-play"]
											},

											{
												"actions": ["update", "upgrads"],
												"systems": ["network", "database", "db"],	---> *
												"tags": ["das"]
											}
										],

									}

								However, in config "annotationDashboard" field also consists of "tags" field. It is a list of dashboard tags which the intelligent module tracks for annotating.

										"annotationDashboard": {
											.
											.
											.
											"tags": ["jobs","das", "cmsweb"]
										}

								So, when we create a cache for Dashboards data based on these tags (here, "tags": ["jobs","das", "cmsweb"]), all dashboards info are saved.
								Thus, we need to find intersection between tags of all these dashboards with those passed with specific services (SSB/GGUS).

										Using example above we can say the intersection result would be,

										["cmsweb-play"] ---> #	since len(["cmsweb-play"]) > 0 returns true
										["das"]			---> *	since len(["cmsweb-play"]) > 0 returns true
							*/

							ifCommonTagsFound := ifTagsIntersect(annotationData.Tags, dashboard.Tags)

							if ifCommonTagsFound == false {
								continue
							}

							var dashboardData models.GrafanaDashboard

							/*Custom tags which consists of intelligence module tag, unique identifier for an alert,
							and tags of all those dashboards where the alert has been annotated.
							*/
							var customTags []string

							customTags = append(customTags, utils.ConfigJSON.AnnotationDashboard.IntelligenceModuleTag) //intelligence module tag (eg. "cmsmon-int")

							lock.RLock()
							val, ok := each.Labels[utils.ConfigJSON.Alerts.UniqueLabel]
							lock.RUnlock()
							if ok {
								//Unique identifier for an alert
								// (eg. ssbNumber for SSB alerts, TicketID for GGUS alerts etc.)
								customTags = append(customTags, val.(string))
							}
							/*
								if val, ok := each.Labels[utils.ConfigJSON.Alerts.UniqueLabel].(string); ok {
									customTags = append(customTags, val) //Unique identifier for an alert (eg. ssbNumber for SSB alerts, TicketID for GGUS alerts etc.)
								}
							*/

							for _, eachTag := range annotationData.Tags {
								customTags = append(customTags, eachTag) //Appending all tags of the dashboard where the alert is going to get annotated.
							}

							dashboardData.DashboardID = dashboard.ID
							dashboardData.Time = each.StartsAt.Unix() * 1000
							dashboardData.TimeEnd = each.EndsAt.Unix() * 1000

							dashboardData.Tags = customTags

							lock.RLock()
							annotationMapLabel := each.Annotations[srv.AnnotationMap.Label]
							lock.RUnlock()

							if val, ok := annotationMapLabel.(string); ok {

								lock.RLock()
								annotationMapURLLabel := each.Annotations[srv.AnnotationMap.URLLabel]
								lock.RUnlock()

								if url, urlOk := annotationMapURLLabel.(string); urlOk {
									dashboardData.Text = srv.Name + ": " + val + "\n" + makeHTMLhref(url)
								} else {
									dashboardData.Text = srv.Name + ": " + val
								}
							}

							dData, err := json.Marshal(dashboardData)
							if err != nil {
								log.Printf("Unable to convert the data into JSON %v, error: %v\n", dashboardData, err)
							}
							if utils.ConfigJSON.Server.Verbose > 0 {
								log.Printf("+++ add annotation to %v", dashboardData.String())
							}
							addAnnotationHelper(dData)
						}
					}
				}
			}
			dataAfterAnnotation <- each
		}
	}()
	return dataAfterAnnotation
}

//makeHTMLhref for making url clickable it needs be tagged with html href attribute that's what this function does
func makeHTMLhref(url string) string {
	return "<a target=_blank href=" + url + ">URL</a>"
}

//ifTagsIntersect for checking intersection between two list of dashboard tags.
func ifTagsIntersect(annotationDashboardTags, allDashboardsTags []string) bool {

	var intersectionResult []string

	hashMap := make(map[string]bool) //a hashmap which helps to reduce the intersection operation from O(n^2) to O(n).

	for _, val := range allDashboardsTags {
		hashMap[val] = true
	}

	for _, each := range annotationDashboardTags {
		if _, ok := hashMap[each]; ok {
			intersectionResult = append(intersectionResult, each)
		}
	}

	return len(intersectionResult) > 0
}

//checkIfAvailable - function for finding if particular keyword is available or not in the given field of Alerts
func checkIfAvailable(data []string, amData models.AmJSON, label string) bool {
	for _, each := range data {

		lock.RLock()
		annotationsLabel := amData.Annotations[label]
		lock.RUnlock()

		if val, ok := annotationsLabel.(string); ok {
			if strings.Contains(strings.ToLower(val), strings.ToLower(each)) {
				return true
			}
		}
	}
	return false
}

//addAnnotationHelper - helper function
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L639
func addAnnotationHelper(data []byte) {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", utils.ConfigJSON.AnnotationDashboard.Token)
	h := []string{"Authorization", bearer}
	headers = append(headers, h)
	h = []string{"Content-Type", "application/json"}
	headers = append(headers, h)

	apiURL := utils.ValidateURL(utils.ConfigJSON.AnnotationDashboard.URL, utils.ConfigJSON.AnnotationDashboard.AnnotationAPI)

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Unable to make request to %s, error: %s", apiURL, err)
		return
	}
	for _, v := range headers {
		if len(v) == 2 {
			req.Header.Add(v[0], v[1])
		}
	}

	if utils.ConfigJSON.Server.Verbose > 1 {
		log.Println("POST", apiURL)
	}
	if utils.ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Println("request: ", string(dump))
		}
	}

	timeout := time.Duration(utils.ConfigJSON.Server.HTTPTimeout) * time.Second
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)

	if err != nil {
		log.Printf("Unable to get response from %s, error: %s", apiURL, err)
		return
	}
	if utils.ConfigJSON.Server.Verbose > 1 {
		dump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Println("response:", string(dump))
		}
	}
	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("response Status:", resp.Status)
		log.Println("response Headers:", resp.Header)
		if utils.ConfigJSON.Server.Verbose > 1 {
			log.Println("response Body:", string(body))
		}
	}

	if resp.StatusCode == http.StatusForbidden {
		utils.ConfigJSON.Server.Testing.AnnotateTestStatus = false
		log.Printf("Unable to annotate the dashboard(s), NO PERMISSION")
		return
	}

	utils.ConfigJSON.Server.Testing.AnnotateTestStatus = true
	log.Printf("Annotation Added Successfully to Grafana Dashboards : data %v", string(data))
}
