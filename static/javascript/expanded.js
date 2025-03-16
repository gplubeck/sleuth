// get all the services for future Server Side Events
document.addEventListener("DOMContentLoaded", function () {
    const serviceContainer = document.getElementById("service-container");

    const serviceCards = document.querySelectorAll(".service-card");

    const services = Array.from(serviceCards).reduce((acc, card) => {
        const id = card.id.split("-")[1]; // get ID
        acc[id] = card;
        return acc;
    },{});

    // taking in id int and json data
    function processEvent(id, data) {
        const card = services[id];
        if (!card) {
            console.error("Service card with ID ${id} not found.");
            return;
        }

        // update status
        const statusIndicator = card.querySelector(".status-indicator");
        statusIndicator.textContent = data.status ? "Online" : "Offline";
        statusIndicator.className = "status-indicator " + (data.status ? "status-online" : "status-offline");

        // Update uptime
        const uptimeElement = card.querySelector(".uptime-info")
        uptimeElement.innerHTML = `<strong>Uptime:</strong> ${data.uptime.toFixed(1)}%`;

        // Update uptime graph
        const graphContainer = card.querySelector(".uptime-graph-container");
        graphContainer.innerHTML = data.uptime_history
            .map(
                uptime =>
                `<div class="uptime-segment ${uptime >= 90 ? "green" : uptime >= 75 ? "yellow" : "red"}" style="flex-grow: 1;"></div>`
            )
            .join("");
        if (service) {
            service.status = data.status.toLowerCase();
            service.uptime = data.uptime;

            // Add the new uptime data with a timestamp
            const currentTime = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
            service.graphData.push({ time: currentTime, value: data.uptime });
        }
    }

    // Connect to the SSE endpoint
    const eventSource = new EventSource("/updates"); // Replace with your backend endpoint

    // Handle incoming updates
    eventSource.onmessage = function (event) {
        try {
            const data = JSON.parse(event.data);
            if (data.id !== undefined) {
                // Update the specific service
                processEvent(data.id, data)
            }
        } catch (err) {
            console.error("Failed to parse SSE data:", err);
        }
    };

    eventSource.onerror = function () {
        console.error("Error connecting to the SSE server.");
    };
});
