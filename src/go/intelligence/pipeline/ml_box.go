package pipeline

import (
	"go/intelligence/models"
)

// Module     : intelligence
// Author     : Rahul Indra <indrarahul2013 AT gmail dot com>
// Created    : Wed, 1 July 2020 11:04:01 GMT
// Description: CERN MONIT infrastructure Intelligence Module

//MlBox Machine Learning predicted Data
//To be implemented
func MlBox(data <-chan models.AmJSON) <-chan models.AmJSON {

	predictedData := make(chan models.AmJSON)
	go func() {
		for d := range data {
			predictedData <- d
		}
		close(predictedData)
	}()
	return predictedData
}
