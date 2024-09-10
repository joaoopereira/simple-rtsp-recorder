const pageTitle = document.getElementById('pageTitle');
const mainTitle = document.getElementById('mainTitle');
const startBtn = document.getElementById("startBtn");
const stopBtn = document.getElementById("stopBtn");
const loadingMessage = document.getElementById("loadingMessage");
const recordingMessage = document.getElementById("recordingMessage");
const recordingsListTitle = document.getElementById("recordingsListTitle");
const recordingsList = document.getElementById("recordingsList");
const elapsedTime = document.getElementById("elapsedTime");
let timerInterval;

function checkStatus() {
  fetch("/status")
    .then((response) => response.json())
    .then((data) => {
      if (data.recording) {
        startBtn.disabled = true;
        stopBtn.disabled = false;
        recordingMessage.classList.remove("hidden");
        elapsedTime.classList.remove("hidden");
        if (data.startTime) {
          startTimer(new Date(data.startTime));
        }
      } else {
        startBtn.disabled = false;
        stopBtn.disabled = true;
        recordingMessage.classList.add("hidden");
        elapsedTime.classList.add("hidden");
        stopTimer();
      }
    })
    .catch((error) => console.error("Error:", error));
}

function startTimer(startTime) {
  if (!startTime) {
    startTime = new Date();
  }
  timerInterval = setInterval(() => {
    const elapsed = Date.now() - startTime.getTime();
    const hours = String(Math.floor(elapsed / 3600000)).padStart(2, "0");
    const minutes = String(Math.floor((elapsed % 3600000) / 60000)).padStart(
      2,
      "0"
    );
    const seconds = String(Math.floor((elapsed % 60000) / 1000)).padStart(
      2,
      "0"
    );
    elapsedTime.textContent = `${hours}:${minutes}:${seconds}`;
  }, 100);
}

function stopTimer() {
  clearInterval(timerInterval);
  elapsedTime.textContent = "Elapsed time: 00:00:00";
}

startBtn.addEventListener("click", () => {
  loadingMessage.classList.remove("hidden");
  fetch("/start")
    .then((response) => response.json())
    .then((data) => {
      loadingMessage.classList.add("hidden");
      startBtn.disabled = true;
      stopBtn.disabled = false;
      recordingMessage.classList.remove("hidden");
      elapsedTime.classList.remove("hidden");
      startTimer(new Date(data.startTime));
    })
    .catch((error) => {
      loadingMessage.classList.add("hidden");
      console.error("Error:", error);
    });
});

stopBtn.addEventListener("click", () => {
  fetch("/stop")
    .then((response) => response.text())
    .then((data) => {
      startBtn.disabled = false;
      stopBtn.disabled = true;
      recordingMessage.classList.add("hidden");
      elapsedTime.classList.add("hidden");
      stopTimer();
      loadRecordings();
    })
    .catch((error) => console.error("Error:", error));
});

function loadRecordings() {
  fetch("/recordings")
    .then((response) => response.json())
    .then((data) => {
      recordingsList.innerHTML = "";
      data.forEach((recording) => {
        const li = document.createElement("li");
        li.classList.add(
          "list-group-item",
          "d-flex",
          "justify-content-between",
          "align-items-center"
        );
        const a = document.createElement("a");
        a.href = `/recordings/${recording}`;
        a.textContent = recording;
        a.download = recording;
        const deleteBtn = document.createElement("button");
        deleteBtn.classList.add("btn", "btn-danger", "btn-sm", "bi", "bi-trash");
        deleteBtn.addEventListener("click", () => {
          fetch(`/recordings/${recording}`, { method: "DELETE" })
            .then((response) => {
              if (response.ok) {
                loadRecordings();
              } else {
                console.error("Failed to delete recording");
              }
            })
            .catch((error) => console.error("Error:", error));
        });
        li.appendChild(a);
        li.appendChild(deleteBtn);
        recordingsList.appendChild(li);
      });
    })
    .catch((error) => console.error("Error:", error));
}

checkStatus();
loadRecordings();

function updateLabels() {
  fetch("/labels")
    .then((response) => response.json())
    .then((data) => {
      pageTitle.innerText = data.LABEL_TITLE;
      mainTitle.innerText = data.LABEL_TITLE;
      startBtn.innerText = data.LABEL_START;
      stopBtn.innerText = data.LABEL_STOP;
      loadingMessage.innerText = data.LABEL_LOADING_MESSAGE + "...";
      recordingMessage.innerText = data.LABEL_RECORDING_MESSAGE + "...";
      recordingsListTitle.innerText = data.LABEL_RECORDINGS_LIST_TITLE
    })
    .catch((error) => console.error("Error:", error));
}

// update labels on page load
updateLabels();
