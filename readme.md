
# 🎥 Simple RTSP Recorder


This is a simple web server that allows you to record files from an RTSP camera stream. It provides an easy-to-use interface for starting and stopping recordings, as well as managing the recorded files. Perfect for quick camera archiving! 🚦


## 🚀 Installation

1. 📥 Clone this repository to your local machine.
2. 📦 Install dependencies: `npm install`
3. 🛠️ Configure the server settings in the `.env` file.
4. ▶️ Start the server: `npm start`


## 🖥️ Usage

1. 🌐 Open your browser and go to `http://localhost:8080`
2. 🎬 Click the **Start** button to begin recording from your RTSP camera stream.
3. ⏹️ Click the **Stop** button to end the recording.
4. 💾 Your recordings will appear in the list and be saved in the output directory.


## ⚙️ Configuration


You can customize the server settings by editing the `.env` file. Here are some of the available options:

* `RTSP_USER` – Username for the RTSP camera stream (optional)
* `RTSP_PASSWORD` – Password for the RTSP camera stream (optional)
* `RTSP_IP` – IP address of the RTSP camera
* `RTSP_PORT` – Port number (usually 554)
* `RTSP_SDP` – SDP file or stream path (e.g. `live1.sdp`)
* `OUTPUT_DIR` – Directory where recordings are saved


## 🤝 Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request. 🙏


## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.
