    // Function to handle SSE
    function handleSSE(event) {
        if (event.data) {
            var eventData = JSON.parse(event.data);
            var card = document.getElementById(eventData.service_name);
            //var card = document.getElementById(eventData.service_name).innerText = eventData.address;
            //grab the elements with respective classes
            var status = card.querySelector(".status");
            var uptime = card.querySelector(".uptime");

            //status online
            if(eventData.status){
                status.classList.replace("offline", "online");
            }else{
                status.classList.replace("online", "offline");
            }

            if(eventData.uptime){
                uptime.innerText = eventData.uptime;
            }
            
        }
    }


    // Open SSE connection
    var eventSource = new EventSource('/updates');
    eventSource.addEventListener('message', handleSSE);
