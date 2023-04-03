// Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
//

// This counter will be used to get the first opening of the page.
// If short url is used, parent URL will be changed to main pages (../)
var GLOBAL_INITIALIZATION_COUNTER = 0
var PAGE_ENDPOINT = "stepchain-detailed"

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
    // Set Stepchain Task name search inputs bar from short url request
    $('#input-sc-task').val(govar_SHORT_URL_REQUEST.searchBuilderRequest.inputScTask);
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
function helperMBytesToHumanReadable(input_bytes) {
    if (input_bytes === 0) {
        return "0.00 MB";
    }
    let e = Math.floor(Math.log(input_bytes) / Math.log(1000));
    return (input_bytes / Math.pow(1000, e)).toFixed(2) + ' ' + 'MGTPE'.charAt(e) + 'B';
}

/*
 * helperFloatPrecision
 */
function helperFloatPrecision(input_num) {
    if (typeof input_num === 'number') {
        return input_num.toFixed(2);
    }
    return input_num;
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
            let _shorturl = window.location.origin + '/' + govar_BASE_EP + '/short-url/' + data
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
    // job_type
    $.fn.dataTable.ext.searchBuilder.conditions.job_type = {
        'Production': {
            conditionName: function (dt, i18n) {return 'Production';},
            isInputValid: function () {return true;},
            init: function () {return;},
            inputValue: function () {return;},
            search: function (value) {return value === 'Production';},
        },
        'Processing': {
            conditionName: function (dt, i18n) {return 'Processing';},
            isInputValid: function () {return true;},
            init: function () {return;},
            inputValue: function () {return;},
            search: function (value) {return value === 'Processing';},
        },
        'Merge': {
            conditionName: function (dt, i18n) {return 'Merge';},
            isInputValid: function () {return true;},
            init: function () {return;},
            inputValue: function () {return;},
            search: function (value) {return value === 'Merge';},
        },
        'LogCollect': {
            conditionName: function (dt, i18n) {return 'LogCollect';},
            isInputValid: function () {return true;},
            init: function () {return;},
            inputValue: function () {return;},
            search: function (value) {return value === 'LogCollect';},
        },
        'Harvesting': {
            conditionName: function (dt, i18n) {return 'Harvesting';},
            isInputValid: function () {return true;},
            init: function () {return;},
            inputValue: function () {return;},
            search: function (value) {return value === 'Harvesting';},
        },
    }

    /*
    * showTaskDetails
    *   Shows detailed individual Task information when green "+" button clicked
    *   It uses a Go controller which returns data from sc_task_(cmsrun|jobtype|site) collections
    *   How it works:
    *     Gets closest "tr" row of the clicked "+" button
    *     Gets workflow value from this "tr" dom element.
    *       This is tricky because, "details-value-task" Class should be added to "Workflow" td(column) dom.
    *       Thanks to DataTables, we can add html class to column elements using "className" in the DT columns array
    *     Gets RseType value from this "tr" dom element same with "Workflow" using className.
    *     Creates a random string to create a temporary div with unique ID for each clicked row.
    *     Stores default background color, changes it to our color and after row is collapsed restore its background color
    *     Sends ajax request to  `../api/rse-detail' API with a JSON request body: {"workflow": .., "type": ..} which defines the workflow.
    *     Gets html response which is sent by the Go controller.
    *        Go controller uses "rse_detail_table.html" template which is also uses DataTables to show pretty table
    *        To not mix current DataTable CSS and "rse_detail_table" CSS, we used random ids and other tricks like "!important;".
    *     And with ".html" function call, response html element showed in the dom.
    *     Thanks to "dt-control", it allows to collapse and expand with green/red color changes.
    */
    function showTaskDetails() {
        var tr = $(this).closest("tr");
        // Contains slashes which can be a problem
        detail_task_name = $(tr).find("td.details-value-task").text()
        detail_step_name = $(tr).find("td.details-value-stepname").text()
        detail_jobtype_name = $(tr).find("td.details-value-jobtype").text()
        random_str = Math.random().toString(36).slice(-10)
        d_class = "details-show"
        row = table.row(tr)
        if (!row.child.isShown()) {
            default_bg_color = $(tr).css("background-color");
            $(tr).addClass(d_class)
            // $(tr).css("background-color", "#CECBCEFF")
            row.child("<div id='" + random_str + "'>loading</div>").show()
            var single_sc_task_detailed_request = {
                "Task": detail_task_name,
                "StepName": detail_step_name,
                "JobType": detail_jobtype_name,
            }
            if (govar_VERBOSITY > 0) {
                console.log(JSON.stringify(single_sc_task_detailed_request))
            }
            $.ajax({
                url: govar_SC_SITE_DETAIL_OF_EACH_CMSRUN_APIEP,
                type: 'post',
                dataType: 'html',
                contentType: 'application/json',
                data: JSON.stringify(single_sc_task_detailed_request),
                success: function (data) {
                    // just to assign random div id
                    $("#" + random_str).html(data);
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
                clearAll: "",
                delete: "Delete",
                title: ""
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
            url: govar_SC_TASK_CMSRUN_JOBTYPE_API_ENDPOINT,
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

                // SearchBuilder set value from INPUT for Task
                sbRequest.inputScTask = $("#input-sc-task").val();
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
            preDefined: {
                criteria: [
                    {
                        data: 'Job Type',
                        origData: 'JobType',
                        condition: 'Merge',
                    },
                ]
            },
            // SearchBuilder customizations to limit conditions: "Task" column not included  they are searched via "input-sc-task"
            columns: [3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24],
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
                    // "int" type will have only "<=", ">=", "between", "null" and "!null" conditions
                    '=': null,
                    '!=': null,
                    '!between': null,
                    '<': null,
                    '>': null,
                },
                array: {
                    // "array" type will have only "="(has_arr_element), "null", "!null" for STRING ARRAY columns
                    '=': {
                        conditionName: 'has_array_element',
                    },
                    '!=': null,
                    'contains': null,
                    'without': null,
                },
            }
        },
        columns: [
            {
                // Details green/red "+"/"-" button
                data: null, className: 'dt-control', orderable: false, defaultContent: '',
                width: "2%"
            },
            {data: "Links", name: 'Links', width: "3%",},
            {
                data: "Task",
                className: "details-value-task",
                searchBuilder: {defaultCondition: "contains"},
                width: "10%",
            },
            {
                data: "StepName",
                name: 'Step Name',
                searchBuilder: {defaultCondition: "contains"},
                className: "details-value-stepname",
                width: "10%",
            },
            {
                data: "JobType",
                name: "Job Type",
                searchBuilderType: 'job_type',
                // Never give default condition, because DataTables sends lots of unnecessary queries each time.
                className: "details-value-jobtype",
                width: "10%",
            },
            {
                data: "CpuEfficiency",
                name: 'Cpu Efficiency',
                searchBuilderType: 'num',
                searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {
                    return type === 'display' ? helperFloatPrecision(data)+'%' : data;
                },
            },
            {data: "NumberOfStep", name: 'Number Of Step', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},},
            {data: "MeanThread", name: 'Mean Thread', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},},
            {data: "MeanStream", name: 'Mean Stream', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},},
            {data: "MeanCpuTimeHr", name: 'Mean Cpu Time Hr', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "TotalCpuTimeHr", name: 'Total Cpu Time Hr', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "MeanJobTimeHr", name: 'Mean Job Time Hr', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "TotalJobTimeHr", name: 'Total Job Time Hr', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "TotalThreadJobTimeHr", name: 'Total Thread Job Time Hr', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "WriteTotalSecs", name: 'Write Total Secs', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "ReadTotalSecs", name: 'Read Total Secs', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "ReadTimePercentage", name: 'Read Time Pct', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "WriteTotalMB", name: 'Write Total MB', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperMBytesToHumanReadable(data) : data;},
            },
            {
                data: "ReadTotalMB", name: 'Read Total MB', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperMBytesToHumanReadable(data) : data;},
            },
            {
                data: "MeanPeakRss", name: 'Mean Peak Rss', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {
                data: "MeanPeakVSize", name: 'Mean Peak VSize', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},
                render: function (data, type, row, meta) {return type === 'display' ? helperFloatPrecision(data) : data;},
            },
            {data: "EraCount", name: 'Era Count', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},},
            {data: "SiteCount", name: 'Site Count', searchBuilderType: 'num', searchBuilder: {defaultCondition: ">="},},
            {data: "AcquisitionEra", name: "Acquisition Era", searchBuilderType: 'array', searchBuilder: {defaultCondition: "="},},
            {data: "Sites", name: "Sites", searchBuilderType: 'array', searchBuilder: {defaultCondition: "="},},
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
                titleAttr: "This table is produced with the WMArchive data in between 6:30AM-7:15AM CERN time",
            },
            {
                className: 'btn btn-light',
                text: '<a href="">Documentation</a>',
                action: function (e, dt, node, config) {
                    //This will send the page to the location specified
                    window.open("https://github.com/dmwm/CMSMonitoring/blob/master/cpueff-goweb/docs/aggregations.md", "_blank");
                }
            },
            {
                className: 'btn btn-light',
                text: '<a href="">GitHub Repo</a>',
                titleAttr: "If you see any bug, please open issue!",
                action: function (e, dt, node, config) {
                    //This will send the page to the location specified
                    window.open("https://github.com/dmwm/CMSMonitoring/tree/master/cpueff-goweb", "_blank");
                }
            }
        ]
    });
    $('#input-sc-task').on('enterKey', function () {
        table.draw();
    });
    $('#input-sc-task').keyup(function (e) {
        if (e.key === 'Enter' || e.keyCode === 13) {
            $(this).trigger("enterKey");
        }
    });
    // Add event listener for opening and closing details
    table.on('click', 'td.dt-control', showTaskDetails);
});
