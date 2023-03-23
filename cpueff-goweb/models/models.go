package models

// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"html/template"
	"strings"
)

// StringArray type used for converting array with comma separated
type StringArray []string

// ---------------------------------------------------------------------- Condor

// CondorWfCpuEff Condor workflow level cpu efficiency entry
type CondorWfCpuEff struct {
	Links                template.HTML `bson:"Links,omitempty"` // not belongs to actual data but required for carrying links to datatable response
	Type                 string        `bson:"Type,omitempty" validate:"required"`
	Workflow             string        `bson:"Workflow,omitempty" validate:"required"`
	WmagentRequestName   string        `bson:"WmagentRequestName,omitempty"`
	CpuEffOutlier        int64         `bson:"CpuEffOutlier,omitempty" validate:"required"`
	CpuEff               float64       `bson:"CpuEff,omitempty" validate:"required"`
	NonEvictionEff       float64       `bson:"NonEvictionEff,omitempty" validate:"required"`
	EvictionAwareEffDiff float64       `bson:"EvictionAwareEffDiff,omitempty" validate:"required"`
	ScheduleEff          float64       `bson:"ScheduleEff,omitempty" validate:"required"`
	Cpus                 int64         `bson:"Cpus,omitempty" validate:"required"`
	CpuTimeHr            float64       `bson:"CpuTimeHr,omitempty" validate:"required"`
	WallClockHr          float64       `bson:"WallClockHr,omitempty" validate:"required"`
	CoreTimeHr           float64       `bson:"CoreTimeHr,omitempty" validate:"required"`
	CommittedCoreHr      float64       `bson:"CommittedCoreHr,omitempty" validate:"required"`
	CommittedWallClockHr float64       `bson:"CommittedWallClockHr,omitempty" validate:"required"`
	WastedCpuTimeHr      float64       `bson:"WastedCpuTimeHr,omitempty" validate:"required"`
	CpuEffT1T2           float64       `bson:"CpuEffT1T2,omitempty" validate:"required"`
	CpusT1T2             int64         `bson:"CpusT1T2,omitempty" validate:"required"`
	CpuTimeHrT1T2        float64       `bson:"CpuTimeHrT1T2,omitempty" validate:"required"`
	WallClockHrT1T2      float64       `bson:"WallClockHrT1T2,omitempty" validate:"required"`
	CoreTimeHrT1T2       float64       `bson:"CoreTimeHrT1T2,omitempty" validate:"required"`
	WastedCpuTimeHrT1T2  float64       `bson:"WastedCpuTimeHrT1T2,omitempty" validate:"required"`
}

// CondorSiteCpuEff Condor site level cpu efficiency entry
type CondorSiteCpuEff struct {
	Links                template.HTML `bson:"Links,omitempty"` // not belongs to actual data but required for carrying links to datatable response
	Type                 string        `bson:"Type,omitempty" validate:"required"`
	Workflow             string        `bson:"Workflow,omitempty" validate:"required"`
	WmagentRequestName   string        `bson:"WmagentRequestName,omitempty"`
	Site                 string        `bson:"Site,omitempty"`
	Tier                 string        `bson:"Tier,omitempty"`
	CpuEffOutlier        int64         `bson:"CpuEffOutlier,omitempty" validate:"required"`
	CpuEff               float64       `bson:"CpuEff,omitempty" validate:"required"`
	NonEvictionEff       float64       `bson:"NonEvictionEff,omitempty" validate:"required"`
	EvictionAwareEffDiff float64       `bson:"EvictionAwareEffDiff,omitempty" validate:"required"`
	ScheduleEff          float64       `bson:"ScheduleEff,omitempty" validate:"required"`
	Cpus                 int64         `bson:"Cpus,omitempty" validate:"required"`
	CpuTimeHr            float64       `bson:"CpuTimeHr,omitempty" validate:"required"`
	WallClockHr          float64       `bson:"WallClockHr,omitempty" validate:"required"`
	CoreTimeHr           float64       `bson:"CoreTimeHr,omitempty" validate:"required"`
	CommittedCoreHr      float64       `bson:"CommittedCoreHr,omitempty" validate:"required"`
	CommittedWallClockHr float64       `bson:"CommittedWallClockHr,omitempty" validate:"required"`
	WastedCpuTimeHr      float64       `bson:"WastedCpuTimeHr,omitempty" validate:"required"`
	Schedds              StringArray   `bson:"Schedds"`
	MaxWmagentJobId      string        `bson:"MaxWmagentJobId"`
}

// CondorMainEachDetailedRequest accepts Type, Workflow, WmagentRequestName, CpuEffOutlier and returns this workflow's site details
type CondorMainEachDetailedRequest struct {
	Type               string `json:"Type" validate:"required" binding:"required"`
	Workflow           string `json:"Workflow" validate:"required" binding:"required"`
	WmagentRequestName string `json:"WmagentRequestName"`
	CpuEffOutlier      int64  `json:"CpuEffOutlier"`
}

// ----------------------------------------------------------------------Stepchain

// StepchainTask Stepchain task level cpu efficiency entry
type StepchainTask struct {
	Links                template.HTML `bson:"Links,omitempty"` // not belongs to actual data but required for carrying links to datatable response
	Task                 string        `bson:"Task,omitempty" validate:"required"`
	CpuEfficiency        float64       `bson:"CpuEfficiency,omitempty" validate:"required"`
	NumberOfStep         float64       `bson:"NumberOfStep,omitempty" validate:"required"`
	MeanThread           float64       `bson:"MeanThread" validate:"required"`
	MeanStream           float64       `bson:"MeanStream,omitempty" validate:"required"`
	MeanCpuTimeHr        float64       `bson:"MeanCpuTimeHr,omitempty" validate:"required"`
	TotalCpuTimeHr       float64       `bson:"TotalCpuTimeHr,omitempty" validate:"required"`
	MeanJobTimeHr        float64       `bson:"MeanJobTimeHr,omitempty" validate:"required"`
	TotalJobTimeHr       float64       `bson:"TotalJobTimeHr,omitempty" validate:"required"`
	TotalThreadJobTimeHr float64       `bson:"TotalThreadJobTimeHr,omitempty"`
	WriteTotalMB         float64       `bson:"WriteTotalMB,omitempty"`
	ReadTotalMB          float64       `bson:"ReadTotalMB,omitempty"`
	MeanPeakRss          float64       `bson:"MeanPeakRss,omitempty"`
	MeanPeakVSize        float64       `bson:"MeanPeakVSize,omitempty"`
	EraCount             float64       `bson:"EraCount,omitempty"`
	SiteCount            float64       `bson:"SiteCount,omitempty"`
	AcquisitionEra       StringArray   `bson:"AcquisitionEra,omitempty"`
}

// StepchainTaskWithCmsrunJobtype Stepchain task, cmsrun, jobtype level cpu efficiency entry
type StepchainTaskWithCmsrunJobtype struct {
	Links                template.HTML `bson:"Links,omitempty"` // not belongs to actual data but required for carrying links to datatable response
	Task                 string        `bson:"Task,omitempty" validate:"required"`
	StepName             string        `bson:"StepName,omitempty" validate:"required"`
	JobType              string        `bson:"JobType,omitempty" validate:"required"`
	CpuEfficiency        float64       `bson:"CpuEfficiency,omitempty" validate:"required"`
	NumberOfStep         float64       `bson:"NumberOfStep,omitempty" validate:"required"`
	MeanThread           float64       `bson:"MeanThread" validate:"required"`
	MeanStream           float64       `bson:"MeanStream,omitempty" validate:"required"`
	MeanCpuTimeHr        float64       `bson:"MeanCpuTimeHr,omitempty" validate:"required"`
	TotalCpuTimeHr       float64       `bson:"TotalCpuTimeHr,omitempty" validate:"required"`
	MeanJobTimeHr        float64       `bson:"MeanJobTimeHr,omitempty" validate:"required"`
	TotalJobTimeHr       float64       `bson:"TotalJobTimeHr,omitempty" validate:"required"`
	TotalThreadJobTimeHr float64       `bson:"TotalThreadJobTimeHr,omitempty"`
	WriteTotalMB         float64       `bson:"WriteTotalMB,omitempty"`
	ReadTotalMB          float64       `bson:"ReadTotalMB,omitempty"`
	MeanPeakRss          float64       `bson:"MeanPeakRss,omitempty"`
	MeanPeakVSize        float64       `bson:"MeanPeakVSize,omitempty"`
	EraCount             float64       `bson:"EraCount,omitempty"`
	SiteCount            float64       `bson:"SiteCount,omitempty"`
	AcquisitionEra       StringArray   `bson:"AcquisitionEra,omitempty"`
	Sites                StringArray   `bson:"Sites,omitempty"`
}

// StepchainTaskWithCmsrunJobtypeSite Stepchain task, cmsrun, jobtype and step level cpu efficiency entry
type StepchainTaskWithCmsrunJobtypeSite struct {
	Links                template.HTML `bson:"Links,omitempty"` // not belongs to actual data but required for carrying links to datatable response
	Task                 string        `bson:"Task,omitempty" validate:"required"`
	StepName             string        `bson:"StepName,omitempty" validate:"required"`
	JobType              string        `bson:"JobType,omitempty" validate:"required"`
	Site                 string        `bson:"Site,omitempty" validate:"required"`
	CpuEfficiency        float64       `bson:"CpuEfficiency,omitempty" validate:"required"`
	NumberOfStep         float64       `bson:"NumberOfStep,omitempty" validate:"required"`
	MeanThread           float64       `bson:"MeanThread" validate:"required"`
	MeanStream           float64       `bson:"MeanStream,omitempty" validate:"required"`
	MeanCpuTimeHr        float64       `bson:"MeanCpuTimeHr,omitempty" validate:"required"`
	TotalCpuTimeHr       float64       `bson:"TotalCpuTimeHr,omitempty" validate:"required"`
	MeanJobTimeHr        float64       `bson:"MeanJobTimeHr,omitempty" validate:"required"`
	TotalJobTimeHr       float64       `bson:"TotalJobTimeHr,omitempty" validate:"required"`
	TotalThreadJobTimeHr float64       `bson:"TotalThreadJobTimeHr,omitempty"`
	WriteTotalMB         float64       `bson:"WriteTotalMB,omitempty"`
	ReadTotalMB          float64       `bson:"ReadTotalMB,omitempty"`
	MeanPeakRss          float64       `bson:"MeanPeakRss,omitempty"`
	MeanPeakVSize        float64       `bson:"MeanPeakVSize,omitempty"`
	EraCount             float64       `bson:"EraCount,omitempty"`
	AcquisitionEra       StringArray   `bson:"AcquisitionEra,omitempty"`
}

// StepchainRowDetailRequest accepts Task name and returns this Task's details
type StepchainRowDetailRequest struct {
	Task     string `json:"Task" validate:"required" binding:"required"`
	StepName string `json:"StepName"` // should not be required
	JobType  string `json:"JobType"`  // should not be required
	Site     string `json:"Site"`     // should not be required
}

// ---------------------------------------------------------------------- Common

// CondorTierEfficiencies tier efficiencies
type CondorTierEfficiencies struct {
	Tier            string  `json:"Tier" validate:"required" binding:"required"`
	Type            string  `json:"Type" validate:"required" binding:"required"`
	TierCpuEff      float64 `json:"TierCpuEff" validate:"required" binding:"required"`
	TierCpus        float64 `json:"TierCpus" validate:"required" binding:"required"`
	TierCpuTimeHr   float64 `json:"TierCpuTimeHr" validate:"required" binding:"required"`
	TierWallClockHr float64 `json:"TierWallClockHr" validate:"required" binding:"required"`
}

// DataSourceTS struct contains data production time means; alas Spark job time period
type DataSourceTS struct {
	Id        primitive.ObjectID `bson:"_id"` // do not send in the json
	StartDate string             `bson:"startDate,omitempty" validate:"required"`
	EndDate   string             `bson:"endDate,omitempty" validate:"required"`
}

// DatatableBaseResponse represents JQuery DataTables response format
type DatatableBaseResponse struct {
	Draw            int         `json:"draw" validate:"required"`            // The value that came in DataTable request, same should be returned
	RecordsTotal    int64       `json:"recordsTotal" validate:"required"`    // Total records which will be showed in the footer
	RecordsFiltered int64       `json:"recordsFiltered" validate:"required"` // Filtered record count which will be showed in the footer
	Data            interface{} `json:"data" validate:"required"`            // Data
}

// ServerInfoResp custom response struct for service information
type ServerInfoResp struct {
	ServiceVersion string `json:"version"`
	Server         string `json:"server"`
}

// MarshalJSON marshal integer unix time to date string in YYYY-MM-DD format
func (t StringArray) MarshalJSON() ([]byte, error) {
	var out string
	if len(t) > 0 {
		out = strings.Join(t, ", ")
	}
	return []byte(`"` + out + `"`), nil
}
