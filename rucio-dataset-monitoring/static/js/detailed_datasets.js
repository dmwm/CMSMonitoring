// This counter will be used to get the first opening of the page.
//     If short url is used, parent URL will be changed to main pages (../)
var GLOBAL_INITIALIZATION_COUNTER = 0
var PAGE_ENDPOINT = "detailed"

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
            "savedState": JSON.parse(global_saved_state_holder),
        }),
        success: function (data) {
            // Call copy clipboard here
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
            processing: "<span class='fa-stack fa-lg'><i class='fa fa-spinner fa-spin fa-stack-2x fa-fw'></i></span>&emsp;Processing ...",
        },
        stateSaveCallback: function (settings, data) {
            // Save the last state to "global_saved_state_holder" variable to use in the short url call
            global_saved_state_holder = JSON.stringify(data)
            if (govar_VERBOSITY > 0) {
                console.log("[stateSaveCallback]");
                console.log(global_saved_state_holder);
            }
        },
        stateLoadCallback: function (settings) {
            // "global_saved_state_setter" given by Go Template, so it restore same state for the shared users.
            if (govar_VERBOSITY > 0) {
                console.log("[stateLoadCallback]");
                console.log(GLOBAL_SAVED_STATE_HOLDER)
                console.log(govar_DETAILED_DATASETS_API_ENDPOINT)
            }
            return JSON.parse(GLOBAL_SAVED_STATE_HOLDER);
        },
        aLengthMenu: [
            [5, 10, 25, 50, 100, 500, 1000, 10000],
            [5, 10, 25, 50, 100, 500, 1000, 10000]
        ],
        ajax: {
            url: govar_DETAILED_DATASETS_API_ENDPOINT,
            method: "POST",
            contentType: 'application/json',
            data: function (d) {
                // What is d: d is main DataTable request which will be sent to API

                // Increment counter, to control first opening of the page
                GLOBAL_INITIALIZATION_COUNTER++;

                // SearchBuilder request holder variable
                var sbRequest = {};
                console.log("TEST")

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
            columns: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17],
            conditions: {
                // "num" type hacking. "num" always parse numeric values, but we need whole string like "10TB"
                // that's why we use "html" type, but it will be used in numeric columns
                // IN SHORT:
                //     - html: float type columns
                //     - date: date type columns
                //     - num:  integer type columns
                html: {
                    // "num" type will have only "starts":Greater Than" and "ends":"Less Than" conditions
                    'starts': {
                        conditionName: 'Greater Than',
                    },
                    'ends': {
                        conditionName: 'Less Than',
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
            {data: "Type", width: "3%"},
            {data: "Dataset"},
            {data: "RSE"},
            {data: "Tier"},
            {data: "C"},
            {data: "RseKind"},
            {
                data: "SizeBytes",
                render: function (data, type, row, meta) {
                    // SearchBuilder will use raw data as search values, but display will be human-readable size format.
                    return type === 'display' ? helperBytesToHumanely(data) : data;
                },
                // orderSequence defines first option when clicked to Columns sort button. We set first as "desc", default was "asc".
                orderSequence: ["desc", "asc"],
                searchBuilderType: 'html',
                width: "7%"
            },
            {
                data: "LastAccess",
                searchBuilderType: 'date',
                searchBuilder: {
                    defaultCondition: "between"
                },
                width: "10%"
            },
            {data: "IsFullyReplicated"},
            {data: "IsLocked"},
            {data: "FilePercentage"},
            {data: "FileCount"},
            {data: "AccessedFileCount"},
            {data: "BlockCount"},
            {data: "ProdLockedBlockCount"},
            {data: "ProdAccounts"},
            {data: "BlockRuleIDs"},
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
});
