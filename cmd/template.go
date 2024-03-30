package main

import (
    "html/template"
    "net/http"
)

func StatusTemplate(store ServiceStore, w http.ResponseWriter) {

    services := store.GetServices()

/*    tmpl := `
    <!DOCTYPE html>
    <html lang="en">
    <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Service Status</title>
    </head>
    <body>
    <h1>Service Status</h1>
    <div id="status-container"></div>

    <script>
    // Function to initialize SSE
    function initSSE() {
        const eventSource = new EventSource("/updates");

        // Event listener for receiving updates
        eventSource.onmessage = function(event) {
            const data = JSON.parse(event.data);
            console.log("Received update:", data);

            // Update or create a card for the received service status
            updateOrCreateCard(data);
        };

        // Event listener for error
        eventSource.onerror = function(error) {
            console.error("SSE Error:", error);
            eventSource.close();
        };
    }

    // Update or create a card for the received service status
    function updateOrCreateCard(serviceStatus) {
        const statusContainer = document.getElementById('status-container');
        let serviceCard = document.getElementById(serviceStatus.service_name);

        // If the card for the service doesn't exist, create it
        if (!serviceCard) {
            serviceCard = document.createElement('div');
            serviceCard.id = serviceStatus.service_name;
            serviceCard.className = 'service-card';
            statusContainer.appendChild(serviceCard);
        }

        // Update the content of the card
        serviceCard.innerHTML = "<h2>${serviceStatus.service_name}</h2> <p>Status: ${serviceStatus.status}</p>";
    }

    // Initialize SSE when the page loads
    window.onload = initSSE;
    </script>

    <style>
    .service-card {
        border: 1px solid #ccc;
        padding: 10px;
        margin-bottom: 10px;
    }
    </style>
    </body>
    </html>

    `
*/
    tmpl := `
    <!DOCTYPE html>
    <html lang="en">
    <head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel = "stylesheet" href="/themes/gruvbox.css">
    <title>Service Status</title>
    </head>
    <body>
    <h1>Service Status</h1>
    
    {{range .}}
    <div class = "card" id={{.Name}}> 
        <div class  =
            {{ if .Status}}
                "status online"
            {{else}}
                "status offline"
            {{end}}
            ></div>
        <div class = "service">{{.Name}}</div>
    </div>
    {{end}}
    </div>

    <script>
    // Function to handle SSE
    function handleSSE(event) {
        if (event.data) {
            var eventData = JSON.parse(event.data);
            var card = document.getElementById(eventData.service_name);
            //var card = document.getElementById(eventData.service_name).innerText = eventData.address;
            var status = card.querySelector(".status");

            //status online
            if(eventData.status){
                status.classList.replace("offline", "online");
            }else{
                status.classList.replace("online", "offline");
            }
        }
    }


    // Open SSE connection
    var eventSource = new EventSource('/updates');
    eventSource.addEventListener('message', handleSSE);
    </script>
    </body>
    </html>
    `
    t, err := template.New("services").Parse(tmpl)
    if err != nil {
        panic (err)
    }

    err = t.Execute(w, services)
    if err != nil {
        panic(err)
    }

}


