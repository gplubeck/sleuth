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
