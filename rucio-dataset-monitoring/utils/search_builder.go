package utils

// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

import (
	"github.com/dmwm/CMSMonitoring/rucio-dataset-monitoring/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"strconv"
	"strings"
	"time"
)

// KB to EB bytes definition, uses x1000
const (
	KB float64 = 1e3
	MB float64 = 1e6
	GB float64 = 1e9
	TB float64 = 1e12
	PB float64 = 1e15
	EB float64 = 1e18
)

// strToInt converts user string to int
func strToInt(str string) int {
	// Remove space
	if strings.Contains(str, " ") {
		str = strings.ReplaceAll(str, " ", "")
	}
	i, err := strconv.Atoi(str)
	if err != nil {
		ErrorLog("cannot parse string to int: %s", str)
		return 0
	}
	InfoLogV1("strToInt %d", i)
	return i
}

// strToFloat converts user string to float64
func strToFloat(str string, typeAbbreviation string) float64 {
	// Remove space
	if strings.Contains(str, " ") {
		str = strings.ReplaceAll(str, " ", "")
	}
	str = strings.ReplaceAll(str, typeAbbreviation, "")
	f, err := strconv.ParseFloat(str, 10)
	if err != nil {
		ErrorLog("cannot parse string to float: %s", str)
		return 0
	}
	InfoLogV1("strToFloat %f", f)
	return f
}

// stringToUnixTime converts string date SB query value to millisecond format which is source data format in MongoDB
func stringToUnixTime(s string) int64 {
	// Parse YYYY-MM-DD
	timeT, err := time.Parse("2006-01-02", s)
	if err != nil {
		ErrorLog("cannot convert string: "+s+"to millisecond: %s", err.Error())
		return 0
	}
	return timeT.Unix()
}

// humanSizeToBytes converts user defined size string to bytes
func humanSizeToBytes(input string) int64 {
	InfoLogV1("humanSizeToBytes input: %s", input)
	input = strings.ToUpper(input)
	switch {
	case strings.Contains(input, "KB"):
		return int64(strToFloat(input, "KB") * KB)
	case strings.Contains(input, "MB"):
		return int64(strToFloat(input, "MB") * MB)
	case strings.Contains(input, "GB"):
		return int64(strToFloat(input, "GB") * GB)
	case strings.Contains(input, "TB"):
		return int64(strToFloat(input, "TB") * TB)
	case strings.Contains(input, "PB"):
		return int64(strToFloat(input, "PB") * PB)
	case strings.Contains(input, "EB"):
		return int64(strToFloat(input, "EB") * EB)
	default:
		return int64(strToFloat(input, ""))
	}
}

// searchBsonSelections creates bson.M using SearchBuilderRequest request
//
//	In main DataTables, there are 3 types in our data: string,html,date,num
//	IN SHORT (DataTables type vs Actual column type):
//	  - string: string type columns
//	  - html  : float type columns
//	  - date  : date type columns
//	  - num   : integer type columns
//	Details:
//	- string:
//	    It only has "contains" condition which is behaved as ReGex.
//	    Since regex is powerful, we don't need starts with, ends with etc. conditions
//	- html:
//	    [IMPORTANT] we use this type for numeric types columns. Because "num" type do not provide whole string like "10TB", only "10"
//	    It has only "starts:Greater Than" and "ends:Less Than", in other words greater than and less than conditions.
//	    They behaved as $lte and $gte
//	    User may provide humanized size definition like "10 TB", it is converted to bytes to use in MongoDB queries
//	- date:
//	    It has only "<", ">", "between", "null", "!null" conditions.
//	    In other words: before, after, between, empty, not empty
//	- num:
//	    It has only "<", ">", "between", "null", "!null" conditions. Used for integer columns like `TotalFileCnt`
func searchBsonSelections(criterion models.SingleCriteria) bson.M {
	switch criterion.Type {
	case "string":
		switch criterion.Condition {
		case "contains": // String type should have only "contains" and regex applies
			return bson.M{criterion.OrigData: primitive.Regex{Pattern: criterion.Value[0], Options: "im"}}
		default:
			ErrorLog(" searchBsonSelections failed type is: %s", criterion.Type)
		}
	case "html":
		switch criterion.Condition {
		case "starts":
			bytesFilter := humanSizeToBytes(criterion.Value[0])
			if bytesFilter == 0 {
				return bson.M{} // If string value is not parsable, return null query
			}
			return bson.M{criterion.OrigData: bson.M{"$gte": bytesFilter}}
		case "ends":
			bytesFilter := humanSizeToBytes(criterion.Value[0])
			if bytesFilter == 0 {
				return bson.M{} // If string value is not parsable, return null query
			}
			log.Println(bytesFilter)
			return bson.M{criterion.OrigData: bson.M{"$lte": bytesFilter}}
		default:
			ErrorLog(" searchBsonSelections failed type is: %s", criterion.Type)
		}
	case "date":
		// For LastAccess column in millisecond format
		switch criterion.Condition {
		case "<":
			return bson.M{criterion.OrigData: bson.M{"$lte": stringToUnixTime(criterion.Value[0])}}
		case ">":
			return bson.M{criterion.OrigData: bson.M{"$gte": stringToUnixTime(criterion.Value[0])}}
		case "between":
			return bson.M{
				"$and": []bson.M{
					{criterion.OrigData: bson.M{"$gte": stringToUnixTime(criterion.Value[0])}},
					{criterion.OrigData: bson.M{"$lte": stringToUnixTime(criterion.Value[1])}},
				}}
		case "null":
			return bson.M{criterion.OrigData: bson.M{"$exists": false}}
		case "!null":
			return bson.M{criterion.OrigData: bson.M{"$exists": true}}
		default:
			ErrorLog(" searchBsonSelections failed type is: %s", criterion.Type)
		}
	case "num":
		switch criterion.Condition {
		case "<":
			return bson.M{criterion.OrigData: bson.M{"$lte": strToInt(criterion.Value[0])}}
		case ">":
			return bson.M{criterion.OrigData: bson.M{"$gte": strToInt(criterion.Value[0])}}
		case "between":
			return bson.M{
				"$and": []bson.M{
					{criterion.OrigData: bson.M{"$gte": strToInt(criterion.Value[0])}},
					{criterion.OrigData: bson.M{"$lte": strToInt(criterion.Value[0])}},
				}}
		case "null":
			return bson.M{criterion.OrigData: bson.M{"$exists": false}}
		case "!null":
			return bson.M{criterion.OrigData: bson.M{"$exists": true}}
		default:
			ErrorLog(" searchBsonSelections failed type is: %s", criterion.Type)
		}
	}
	return bson.M{}
}

// GetSearchBuilderBson iterates over all criteria(s) and creates "AND"/"OR" bson.M query
func GetSearchBuilderBson(sb *models.SearchBuilderRequest) bson.M {
	var andQuery []bson.M
	for _, condition := range sb.Criteria {
		andQuery = append(andQuery, searchBsonSelections(condition))
	}
	switch sb.Logic {
	case "AND":
		return bson.M{"$and": andQuery}
	case "OR":
		return bson.M{"$or": andQuery}
	default:
		return bson.M{}
	}
}
