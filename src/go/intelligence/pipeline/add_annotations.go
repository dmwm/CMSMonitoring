package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/dmwm/CMSMonitoring/src/go/intelligence/models"
	"github.com/dmwm/CMSMonitoring/src/go/intelligence/utils"
	"log"
	"net/http"
	"strings"
	"time"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CMS MONIT infrastructure Intelligence Module

// AddAnnotation - function for adding annotations to dashboards
func AddAnnotation(data <-chan models.AmJSON) <-chan models.AmJSON {

	if utils.ConfigJSON.Server.Verbose > 0 {
		log.Println("AddAnnotation step")
	}

	dataAfterAnnotation := make(chan models.AmJSON)
	ptr := &utils.DCache
	ptr.UpdateDashboardCache()

	go func() {
		defer close(dataAfterAnnotation)

		for each := range data {
			if utils.ConfigJSON.Server.Verbose > 0 {
				log.Println(each.String())
			}
			var srv models.Service
			ifServiceFound := false

			for _, service := range utils.ConfigJSON.Services {
				slabel, ok := utils.Get(each.Labels, utils.ConfigJSON.Alerts.ServiceLabel)
				if ok && slabel == service.Name {
					srv = service
					ifServiceFound = true
					break
				}
			}
			if utils.ConfigJSON.Server.Verbose > 2 {
				log.Printf("service found %v %+v\n", ifServiceFound, srv)
			}

			if ifServiceFound {

				for _, annotationData := range srv.AnnotationMap.AnnotationsData {

					//If action keywords match in alerts
					ifActionFound := checkIfAvailable(annotationData.Actions, each, srv.AnnotationMap.Label)
					//If system keywords match in alerts
					ifSystemFound := checkIfAvailable(annotationData.Systems, each, srv.AnnotationMap.Label)

					if utils.ConfigJSON.Server.Verbose > 1 {
						log.Printf("annotation data: actions=%+v systems=%+v\n", ifActionFound, ifSystemFound)
					}

					if ifActionFound && ifSystemFound {
						if len(utils.DCache.Dashboards) == 0 {
							log.Println("No annotation dashboards is provided, annotation will be skipped")
						}
						if utils.ConfigJSON.Server.Verbose > 2 {
							log.Println("dashboards", utils.DCache.Dashboards)
						}

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

							if utils.ConfigJSON.Server.Verbose > 1 {
								log.Printf("add annotation common tags: %v, annotation tags %v, dashbaord tags %v\n", ifCommonTagsFound, annotationData.Tags, dashboard.Tags)
							}

							if ifCommonTagsFound == false {
								continue
							}

							var dashboardData models.GrafanaDashboard

							// Custom tags which consists of intelligence module
							// tag, unique identifier for an alert, and tags of
							// all those dashboards where the alert has been annotated.
							var customTags []string

							//intelligence module tag (eg. "cmsmon-int")
							customTags = append(customTags, utils.ConfigJSON.AnnotationDashboard.IntelligenceModuleTag)

							if val, ok := utils.Get(each.Labels, utils.ConfigJSON.Alerts.UniqueLabel); ok {
								// Unique identifier for an alert
								// (eg. ssbNumber for SSB alerts, TicketID for GGUS alerts etc.)
								customTags = append(customTags, val)
							}

							for _, eachTag := range annotationData.Tags {
								// Appending all tags of the dashboard where the alert is going to get annotated.
								customTags = append(customTags, eachTag)
							}

							dashboardData.DashboardID = dashboard.ID
							dashboardData.Time = each.StartsAt.Unix() * 1000
							dashboardData.TimeEnd = each.EndsAt.Unix() * 1000
							dashboardData.Tags = customTags

							if val, ok := utils.Get(each.Annotations, srv.AnnotationMap.Label); ok {
								dashboardData.Text = srv.Name + ": " + val
								if url, urlOk := utils.Get(each.Annotations, srv.AnnotationMap.URLLabel); urlOk {
									dashboardData.Text = srv.Name + ": " + val + "\n" + makeHTMLhref(url)
								}
							}
							addAnnotationHelper(dashboardData)
						}
					}
				}
			}
			dataAfterAnnotation <- each
		}
	}()
	return dataAfterAnnotation
}

// makeHTMLhref for making url clickable it needs be tagged with html href attribute that's what this function does
func makeHTMLhref(url string) string {
	return "<a target=_blank href=" + url + ">URL</a>"
}

// ifTagsIntersect for checking intersection between two list of dashboard tags.
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

// checkIfAvailable - function for finding if particular keyword is available or not in the given field of Alerts
func checkIfAvailable(data []string, amData models.AmJSON, label string) bool {
	for _, each := range data {
		if val, ok := utils.Get(amData.Annotations, label); ok {
			if strings.Contains(strings.ToLower(val), strings.ToLower(each)) {
				return true
			}
		}
	}
	return false
}

// addAnnotationHelper - helper function
// The following block of code was taken from
// https://github.com/dmwm/CMSMonitoring/blob/master/src/go/MONIT/monit.go#L639
func addAnnotationHelper(d models.GrafanaDashboard) {
	var headers [][]string
	bearer := fmt.Sprintf("Bearer %s", utils.ConfigJSON.AnnotationDashboard.Token)
	headers = append(headers, []string{"Authorization", bearer})
	headers = append(headers, []string{"Content-Type", "application/json"})

	apiURL := utils.ValidateURL(utils.ConfigJSON.AnnotationDashboard.URL, utils.ConfigJSON.AnnotationDashboard.AnnotationAPI)
	dData, err := json.Marshal(d)
	if err != nil {
		log.Printf("Unable to convert the data into JSON %v, error: %v\n", d, err)
		return
	}
	resp := utils.HttpCall("POST", apiURL, headers, bytes.NewBuffer(dData))
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		utils.ConfigJSON.Server.Testing.AnnotateTestStatus = false
		log.Printf("Unable to annotate the dashboard(s), NO PERMISSION")
		return
	}

	utils.ConfigJSON.Server.Testing.AnnotateTestStatus = true
	tms := time.Unix(d.Time/1000, 0)    // dashboard time is in milliseconds
	tme := time.Unix(d.TimeEnd/1000, 0) // dashboard time is in milliseconds
	log.Printf("add annotation '%s' from %v to %v to dashboard %v tags %v", d.Text, tms, tme, d.DashboardID, d.Tags)
}
