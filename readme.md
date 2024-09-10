# Simple RTSP Recorder

This is a simple web server that allows you to record files from an RTSP camera stream. It provides an easy-to-use interface for starting and stopping recordings, as well as managing the recorded files.

## Installation

1. Clone this repository to your local machine.
2. Install the required dependencies by running `npm install`.
3. Configure the server settings in the `.env` file.
4. Start the server by running `npm start`.

## Usage

1. Access the web interface by navigating to `http://localhost:8080` in your browser.
2. Enter the RTSP camera stream URL and click on the "Start Recording" button.
3. To stop the recording, click on the "Stop Recording" button.
4. The recorded files will be saved in the specified output directory.

## Configuration

You can customize the server settings by modifying the `.env` file. Here are some of the available options:

- `RTSP_USER`: The username for the RTSP camera stream.
- `RTSP_PASSWORD`: The password for the RTSP camera stream.
- `RTSP_IP`: The IP address of the RTSP camera stream.
- `RTSP_PORT`: The port number of the RTSP camera stream.
- `RTSP_SDP`: The SDP file for the RTSP camera stream.
- `OUTPUT_DIR`: The directory where the recorded files will be saved.

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.




































- `outputDir`: The directory where the recorded files will be saved.
- `maxDuration`: The maximum duration (in seconds) for each recording.

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please open an issue or submit a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.
