// Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//

// This counter will be used to get the first opening of the page.
// If short url is used, parent URL will be changed to main pages (../)
var GLOBAL_INITIALIZATION_COUNTER = 0
var PAGE_ENDPOINT = "main"

// Global variable to catch latest datatables request and set if provided in short url
var GLOBAL_DT_REQUEST_HOLDER = null;

// Global variable to catch the latest DT dom state as json(i.e.: SearchBuilder view) and set if provided in short url
var GLOBAL_SAVED_STATE_HOLDER = null;

// If short url, get saved state from the GoLang controller Template
if (govar_IS_SHORT_URL === true) {
    GLOBAL_SAVED_STATE_HOLDER = JSON.stringify(govar_DT_SAVED_STATE);
    if (govar_VERBOSITY > 0) {
        console.log("[govar_IS_SHORT_URL true]")
        console.log(GLOBAL_SAVED_STATE_HOLDER)
    }
    // Set Main dataset search input bar from short url request
    $('#input-dataset').val(govar_SHORT_URL_REQUEST.searchBuilderRequest.inputDataset);
}

// ---------~~~~~~~~~  FUNCTIONS AND MAIN DEFINITIONS ~~~~~~~~~---------
// Required to show text on hover, used for source date(timestamp) hover setting
$(function () {
    $('[data-toggle="tooltip"]').tooltip()
})

/*
 * helperBytesToHumanely
 *   Converts bytes to human-readable format KB,MB,GB,TB,PB,EB(max) with 2 decimals
 *   If you want to implement ZB, YB, etc., you need to modify Go controller that parses these values
 *   Size data stored as bytes in integer format.
 */
function helperBytesToHumanely(input_bytes) {
    if (input_bytes === 0) {
        return "0.00 B";
    }
    let e = Math.floor(Math.log(input_bytes) / Math.log(1000));
    return (input_bytes / Math.pow(1000, e)).toFixed(2) + ' ' + ' KMGTPE'.charAt(e) + 'B';
}

/*
 * helperCopyToClipboard
 *   Helper function to copy given data to clipboard
 */
function helperCopyToClipboard(message) {
    let textArea = document.createElement("textarea");
    textArea.value = message;
    textArea.style.opacity = "0";
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    try {
        let successful = document.execCommand('copy');
        // var msg = successful ? 'successful' : 'unsuccessful';
        alert('Copying successful:' + message);
    } catch (err) {
        alert('Unable to copy value , error : ' + err.message);
    }
    document.body.removeChild(textArea);
}

/*
 * getShortUrl
 *   Mainly it gets short url from '../api/short-url' API call when "Copy state link to clipboard" button clicked.
 *     short url creation requires a request which should be a JSON object {"dtRequest": .., "savedState": .. }
 *     "dtRequest" holds saved last fired datatables request
 *     "savedState" holds last state(HTML, CSS, DataTables, SearchBuilder, page, sort order, etc.) ...
 *      ... which is important to share the exact state(page number, sort order, etc).
 *     Go controller gets this request, converts to json string and gets its MD5 hash to not save same requests again and again
 *     Go controller returns MD5 hash id to JS ajax call below
 *     And function writes MD5 hash id to clipboard with page origin url
 */
function getShortUrl() {
    $.ajax({
        url: govar_SHORT_URL_API_ENDPOINT,
        type: 'post',
        contentType: 'application/json',
        data: JSON.stringify({
            "page": PAGE_ENDPOINT,
            "dtRequest": JSON.parse(GLOBAL_DT_REQUEST_HOLDER),
            "savedState": JSON.parse(GLOBAL_SAVED_STATE_HOLDER),
        }),
        success: function (data) {
            let _shorturl = window.location.origin + '/'+ govar_BASE_EP + '/short-url/' + data
            if (govar_VERBOSITY > 0) {
                console.log("Short URL: " + _shorturl)
            }
            // Call copy clipboard here
            helperCopyToClipboard(_shorturl);
        },
        error: function () {
            alert('Copy failed');
        }
    });
}

/*
 * DataTables engine is starting to run ...
 */
$(document).ready(function () {
    // CUSTOM SEARCH-BUILDER TYPES
    // tape_disk
    $.fn.dataTable.ext.searchBuilder.conditions.tape_disk = {
        'TAPE': {
            conditionName: function (dt, i18n) { return 'TAPE'; },
            isInputValid: function () { return true; },
            init: function () { return; },
            inputValue: function () { return; },
            search: function (value) { return value === 'TAPE'; },
        },
        'DISK': {
            conditionName: function (dt, i18n) { return 'DISK'; },
            isInputValid: function () { return false; },
            init: function () { return; },
            inputValue: function () { return; },
            search: function (value) { return value === 'DISK'; },
        },
    }

    /*
    * showDatasetDetails
    *   Shows detailed individual dataset information when green "+" button clicked
    *   It uses a Go controller which returns data from detailed_datasets collection
    *   How it works:
    *     Gets closest "tr" row of the clicked "+" button
    *     Gets dataset value from this "tr" dom element.
    *       This is tricky because, "details-value-dataset" Class should be added to "Dataset" td(column) dom.
    *       Thanks to DataTables, we can add html class to column elements using "className" in the DT columns array
    *     Gets RseType value from this "tr" dom element same with "Dataset" using className.
    *     Creates a random string to create a temporary div with unique ID for each clicked row.
    *     Stores default background color, changes it to our color and after row is collapsed restore its background color
    *     Sends ajax request to  `../api/rse-detail' API with a JSON request body: {"dataset": .., "type": ..} which defines the dataset.
    *     Gets html response which is sent by the Go controller.
    *        Go controller uses "rse_detail_table.html" template which is also uses DataTables to show pretty table
    *        To not mix current DataTable CSS and "rse_detail_table" CSS, we used random ids and other tricks like "!important;".
    *     And with ".html" function call, response html element showed in the dom.
    *     Thanks to "dt-control", it allows to collapse and expand with green/red color changes.
    */
    function showDatasetDetails() {
        var tr = $(this).closest("tr");
        dataset_name = $(tr).find("td.details-value-dataset").text()
        type_name = $(tr).find("td.details-value-rse-type").text()
        random_str = Math.random().toString(36).slice(-10)
        d_class = "details-show"
        row = table.row(tr)
        if (!row.child.isShown()) {
            default_bg_color = $(tr).css("background-color");
            $(tr).addClass(d_class)
            // $(tr).css("background-color", "#CECBCEFF")
            row.child("<div id='" + type_name + random_str + "'>loading</div>").show()
            var single_dataset_request = {
                "dataset": dataset_name,
                "type": type_name
            }
            //console.log(JSON.stringify(single_dataset_request))
            $.ajax({
                url: govar_EACH_RSE_DETAILS_API_ENDPOINT,
                type: 'post',
                dataType: 'html',
                contentType: 'application/json',
                data: JSON.stringify(single_dataset_request),
                success: function (data) {
                    // just to assign random div id
                    $("#" + type_name + random_str).html(data);
                },
            });
            tr.addClass('selected-row-color')
        } else {
            $(tr).removeClass(d_class)
            $(tr).css("background-color", default_bg_color)
            row.child.hide()
        }
    }

    /*
     * DataTables engine is running at full speed ...
     *    All below settings are important, even a single character.
     *    Please read lots of nice documentation, examples, blogs, great community Q&A in https://datatables.net/
     *      to understand what it is happening.
     */
    var table = $('.go-datatable').DataTable({
        serverSide: true,
        processing: true,
        stateSave: true,
        stateDuration: 0,
        select: false,
        scrollX: false,
        pageLength: 10,
        order: [], // no initial sorting
        dom: "iBQplrt", // no main search ("f"), just individual column search
        language: {
            searchBuilder: {
                clearAll: 'Reset',
                delete: 'Delete',
            },
            processing: "Processing ...",
        },
        stateSaveCallback: function (settings, data) {
            // Save the last state to "GLOBAL_SAVED_STATE_HOLDER" variable to use in the short url call
            GLOBAL_SAVED_STATE_HOLDER = JSON.stringify(data)
            if (govar_VERBOSITY > 0) {
                console.log("[stateSaveCallback]");
                console.log(GLOBAL_SAVED_STATE_HOLDER);
            }
        },
        stateLoadCallback: function (settings) {
            // "global_saved_state_setter" given by Go Template, so it restore same state for the shared users.
            if (govar_VERBOSITY > 0) {
                console.log("[stateLoadCallback]");
                console.log(GLOBAL_SAVED_STATE_HOLDER)
            }
            return JSON.parse(GLOBAL_SAVED_STATE_HOLDER);
        },
        // stateLoadParams: function (settings, data) {
        // },
        // stateSaveParams: function (setting, data){
        // },
        aLengthMenu: [
            [5, 10, 25, 50, 100, 500, 1000, 10000],
            [5, 10, 25, 50, 100, 500, 1000, 10000]
        ],
        ajax: {
            url: govar_MAIN_DATASETS_API_ENDPOINT,
            method: "POST",
            contentType: 'application/json',
            data: function (d) {
                // What is d: d is main DataTable request which will be sent to API

                // Increment counter, to control first opening of the page
                GLOBAL_INITIALIZATION_COUNTER++;

                // SearchBuilder request holder variable
                var sbRequest = {};

                // Check if user created a search builder query using SB Conditions
                try {
                    sbRequest = table.searchBuilder.getDetails(true);
                    if (govar_VERBOSITY > 0) {
                        console.log("[DT-ajax: sbRequest]")
                        console.log(JSON.stringify(sbRequest));
                    }
                } catch (error) {
                    // User did not create SearchBuilder query, so set the request holder variable as null
                    sbRequest = {};
                }
                sbRequest.inputDataset = $("#input-dataset").val();
                // Add SearchBuilder JSON object to DataTable main request
                d.searchBuilderRequest = sbRequest;

                // Check if short url is used
                if (govar_IS_SHORT_URL === true) {
                    // Set datatable request from saved short-url request using go template
                    // So we modify the DataTable request with the shared url request stored and fetched from MongoDB
                    d = govar_SHORT_URL_REQUEST;
                    GLOBAL_DT_REQUEST_HOLDER = JSON.stringify(d);

                    // Since all operations for short url is done, we need to set it to false
                    govar_IS_SHORT_URL = false;

                    // Change origin url to main page for further request of users who used short url
                    if (GLOBAL_INITIALIZATION_COUNTER === 1) {
                        window.history.pushState('', 'Title', '../' + PAGE_ENDPOINT);
                    }
                } else {
                    GLOBAL_DT_REQUEST_HOLDER = JSON.stringify(d);
                }
                if (govar_VERBOSITY > 0) {
                    console.log("--- MAIN ajax global_dt_request_holder ----");
                    console.log(GLOBAL_DT_REQUEST_HOLDER);
                }
                return GLOBAL_DT_REQUEST_HOLDER;
            },
            dataType: "json",
        },
        search: {
            // Run search on enter effectively, works in SearchBuilder too
            return: true
        },
        searchBuilder: {
            depthLimit: 1,
            // SearchBuilder customizations to limit conditions: "datasets" column not included because it is searched via "input-dataset"
            columns: [1, 3, 4, 5, 6, 7, 8, 9, 10, 11],
            // greyscale: true,
            // Sends additional query, that's why disabled.
            // preDefined: {
            //     criteria: [
            //         {
            //             data: 'Rse Type',
            //             origData: 'RseType',
            //             condition: 'contains',
            //             value: ["DISK"]
            //         },
            //     ]
            // },
            conditions: {
                // "num" type hacking. "num" always parse numeric values, but we need whole string like "10TB"
                // that's why we use "html" type, but it will be used in numeric columns
                // IN SHORT:
                //     - html: float type columns
                //     - date: date type columns
                //     - num:  integer type columns
                //     - tape_disk: Rse Type column
                tape_disk: {},
                html: {
                    // "num" type will have only "starts":GreaterThan" and "ends":"LessThan" conditions
                    'starts': {
                        conditionName: 'GreaterThan',
                    },
                    'ends': {
                        conditionName: 'LessThan',
                    },
                    '=': null,
                    '!=': null,
                    '!starts': null,
                    '!contains': null,
                    '!ends': null,
                    'null': null,
                    '!null': null,
                    'contains': null,
                },
                string: {
                    // "string" type will have only "contains" condition which will use Regex search in Go controller.
                    '=': null,
                    '!=': null,
                    'starts': null,
                    '!starts': null,
                    '!contains': null,
                    'ends': null,
                    '!ends': null,
                    'null': null,
                    '!null': null,
                    'contains': {
                        conditionName: 'regex',
                    },
                },
                date: {
                    // "date" type will have only ">", "<", "between", "null" and "!null" conditions
                    '>': null,
                    '<': null,
                    '=': null,
                    '!=': null,
                    '!between': null,
                },
                num: {
                    // "int" type will have only ">", "<", "between", "null" and "!null" conditions
                    '=': null,
                    '!=': null,
                    '!between': null,
                    '<=': null,
                    '>=': null,
                }
            }
        },
        columns: [
            {
                // Details green/red "+"/"-" button
                data: null, className: 'dt-control', orderable: false, defaultContent: '',
                width: "2%"
            },
            {
                data: "RseType",
                className: "details-value-rse-type",
                name: 'Rse Type',
                searchBuilderType: 'tape_disk',
                width: "3%"
            },
            {
                data: "Dataset",
                className: "details-value-dataset",
                //width: "20%"
            },
            {
                data: "LastAccess",
                searchBuilderType: 'date',
                searchBuilder: {
                    defaultCondition: "between"
                },
                width: "10%"
            },
            {
                data: "Max",
                render: function (data, type, row, meta) {
                    // SearchBuilder will use raw data as search values, but display will be human-readable size format.
                    return type === 'display' ? helperBytesToHumanely(data) : data;
                },
                // orderSequence defines first option when clicked to Columns sort button. We set first as "desc", default was "asc".
                orderSequence: ["desc", "asc"],
                searchBuilderType: 'html',
                searchBuilder: {defaultCondition: "starts"},
                width: "7%"
            },
            {
                data: "Min",
                render: function (data, type, row, meta) {
                    // SearchBuilder will use raw data as search values, but display will be human-readable size format.
                    return type === 'display' ? helperBytesToHumanely(data) : data;
                },
                orderSequence: ["desc", "asc"],
                searchBuilderType: 'html',
                searchBuilder: {defaultCondition: "starts"},
                width: "7%"
            },
            {
                data: "Avg",
                render: function (data, type, row, meta) {
                    // SearchBuilder will use raw data as search values, but display will be human-readable size format.
                    return type === 'display' ? helperBytesToHumanely(data) : data;
                },
                orderSequence: ["desc", "asc"],
                searchBuilderType: 'html',
                searchBuilder: {defaultCondition: "starts"},
                width: "7%"
            },
            {
                data: "Sum",
                render: function (data, type, row, meta) {
                    // SearchBuilder will use raw data as search values, but display will be human-readable size format.
                    return type === 'display' ? helperBytesToHumanely(data) : data;
                },
                orderSequence: ["desc", "asc"],
                searchBuilderType: 'html',
                searchBuilder: {defaultCondition: "starts"},
                width: "7%"
            },
            {
                data: "RealSize",
                render: function (data, type, row, meta) {
                    // SearchBuilder will use raw data as search values, but display will be human-readable size format.
                    return type === 'display' ? helperBytesToHumanely(data) : data;
                },
                orderSequence: ["desc", "asc"],
                searchBuilderType: 'html',
                searchBuilder: {defaultCondition: "starts"},
                width: "5%"
            },
            {
                data: "TotalFileCnt",
                name: 'FileCnt',
                searchBuilderType: 'num',
                searchBuilder: {defaultCondition: ">"},
                width: "3%"
            },
            {
                data: "RSEs",
                className: "rses-style",
                searchBuilder: {
                    defaultCondition: "contains"
                },
                width: "20%"
            }
        ],
        buttons: [
            {
                extend: 'copyHtml5',
                className: "btn btn-primary",
                exportOptions: {
                    columns: ':visible'
                }
            },
            {
                extend: 'excelHtml5',
                className: "btn btn-primary",
                exportOptions: {
                    columns: ':visible'
                }
            },
            {
                extend: 'pdfHtml5',
                className: "btn btn-primary",
                exportOptions: {
                    columns: ':visible'
                }
            },
            {
                extend: 'colvis',
                className: "btn btn-primary",
            },
            {
                text: 'Copy state link to clipboard',
                className: "btn btn-primary",
                action: function (e, dt, node, config) {
                    getShortUrl();
                    // navigator.clipboard.writeText is not working because of https://developer.mozilla.org/en-US/docs/Web/Security/Secure_Contexts
                }
            },
            {
                className: 'btn btn-light glyphicon glyphicon-time',
                text: 'DataTimestamp: ' + govar_SOURCE_DATE,
                titleAttr: "This table is produced with the data of Rucio and DBS sqoop dumps runs in between 6:30AM-7:15AM CERN time",
            },
            {
                className: 'btn btn-light',
                text: '<a href="">GitHub Repo</a>',
                titleAttr: "If you see any bug, please open issue!",
                action: function (e, dt, node, config) {
                    //This will send the page to the location specified
                    window.open("https://github.com/dmwm/CMSMonitoring/tree/master/rucio-dataset-monitoring", "_blank");
                }
            },
            {
                className: 'btn btn-light',
                text: '<a href="">Examples</a>',
                action: function (e, dt, node, config) {
                    //This will send the page to the location specified
                    window.open("https://github.com/dmwm/CMSMonitoring/blob/master/rucio-dataset-monitoring/docs/example_query.md", "_blank");
                }
            }
        ]
    });
    $('#input-dataset').on('enterKey', function () {
        table.draw();
    });
    $('#input-dataset').keyup(function (e) {
        if (e.key === 'Enter' || e.keyCode === 13) {
            $(this).trigger("enterKey");
        }
    });
    // Add event listener for opening and closing details
    table.on('click', 'td.dt-control', showDatasetDetails);
});
