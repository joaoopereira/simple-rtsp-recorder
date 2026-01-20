
# 🎥 Simple RTSP Recorder


This is a simple web server that allows you to record files from an RTSP camera stream. It provides an easy-to-use interface for starting and stopping recordings, as well as managing the recorded files. Perfect for quick camera archiving! 🚦


## 🚀 Installation

### Quick Install (Windows Production)

**One-liner installation** (Run PowerShell as Administrator):
```powershell
irm https://raw.githubusercontent.com/joaoopereira/simple-rtsp-recorder/development/install.ps1 | iex
```

This will:
- Download the latest release from GitHub
- Install to the current directory
- Set up a Windows service
- Create a default configuration file

After installation:
1. Edit `prod.env` with your camera settings
2. The service will be running automatically
3. Access the web interface at `http://localhost:8080`

### Development Install

1. 📥 Clone this repository to your local machine
2. 📦 Install dependencies: `npm install`
3. 🛠️ Configure the server settings in the `.env` file
4. ▶️ Start the server: `npm start`

### Manual Windows Installation

1. Download the latest release from the [Releases page](https://github.com/joaoopereira/simple-rtsp-recorder/releases)
2. Extract `simple-rtsp-recorder-win-x64.zip` to your desired location
3. Edit `prod.env` with your camera settings
4. Run `Install-Service.ps1` as Administrator to install as a Windows service
5. Access the web interface at `http://localhost:8080`


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

## 🔧 Building Releases

To create a new release:

```bash
# Bump version and create a tag
npm run bump

# Push the tag to trigger GitHub Actions build
git push --follow-tags
```

This will automatically build the Windows binary and publish it as a GitHub release.


## 📄 License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.
