const express = require("express");
const ffmpeg = require("fluent-ffmpeg");
const path = require("path");
const fs = require("fs");
const dotenv = require("dotenv");
const winston = require("winston");

// Configure dotenv
dotenv.config({
    path: path.resolve(__dirname, process.env.ENV_FILE || 'prod.env')
});

// Configure winston
const logger = winston.createLogger({
    level: 'info',
    format: winston.format.combine(
        winston.format.timestamp(),
        winston.format.printf(({ timestamp, level, message }) => {
            return `${timestamp} [${level.toUpperCase()}]: ${message}`;
        })
    ),
    transports: [
        new winston.transports.File({ filename: 'server.log' }),
        new winston.transports.Console()
    ]
});

const app = express();
const port = process.env.PORT || 8080;

let recordingProcess = null;
let recordingStartTime = null;

app.use(express.static("public"));

app.get("/start", (req, res) => {
  if (recordingProcess) {
    logger.warn("Attempt to start recording while another is in progress");
    return res.status(400).send("Recording already in progress");
  }

  const rtspUrl = `rtsp://${process.env.RTSP_USER}:${process.env.RTSP_PASSWORD}@${process.env.RTSP_IP}:${process.env.RTSP_PORT}/${process.env.RTSP_SDP}`;
  const outputDir = process.env.OUTPUT_DIR;
  const now = new Date();
  const year = now.getFullYear();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  const hour = String(now.getHours()).padStart(2, "0");
  const minute = String(now.getMinutes()).padStart(2, "0");
  const second = String(now.getSeconds()).padStart(2, "0");
  const timestamp = `${year}-${month}-${day}_${hour}${minute}${second}`;
  const outputPath = path.join(outputDir, `${timestamp}.mp4`);

  // Start recording if the stream is available
  recordingProcess = ffmpeg(rtspUrl)
    .outputOptions(["-vcodec copy", "-acodec copy"])
    .save(outputPath)
    .on("start", (output) => {
      logger.info("Recording started");
      recordingStartTime = now; // Store the start time
      logger.info(output);
      res.json({
        message: "Recording started",
        startTime: recordingStartTime.toISOString(),
      });
    })
    .on("end", (output) => {
      logger.info("Recording finished");
      logger.info(output);
      recordingProcess = null;
      recordingStartTime = null; // Reset the start time
    })
    .on("error", (err) => {
      logger.error(`FFmpeg error: ${err.message}`);
      recordingProcess = null;
      recordingStartTime = null; // Reset the start time
    });
});

app.get("/stop", (req, res) => {
  if (!recordingProcess) {
    logger.warn("Attempt to stop recording when no recording is in progress");
    return res.status(400).send("No recording in progress");
  }

  recordingProcess.kill("SIGINT");
  recordingProcess = null;
  recordingStartTime = null; // Reset the start time
  logger.info("Recording stopped");
  res.send("Recording stopped");
});

app.get("/recordings", (req, res) => {
  const outputDir = process.env.OUTPUT_DIR;
  fs.readdir(outputDir, (err, files) => {
    if (err) {
      logger.error("Unable to list recordings");
      return res.status(500).send("Unable to list recordings");
    }

    const recordings = files
      .filter((file) => file.endsWith(".mp4"))
      .map((file) => ({
        name: file,
        createdAt: fs.statSync(path.join(outputDir, file)).birthtime,
      }))
      .sort((a, b) => b.createdAt - a.createdAt)
      .map((record) => record.name);

    res.json(recordings);
  });
});

app.get("/recordings/:filename", (req, res) => {
  const outputDir = process.env.OUTPUT_DIR;
  const filePath = path.join(outputDir, req.params.filename);
  res.download(filePath, (err) => {
    if (err) {
      logger.error(`Failed to download file: ${req.params.filename}`);
      res.status(500).send("Failed to download file");
    }
  });
});

// Endpoint to delete a recording
app.delete('/recordings/:filename', (req, res) => {
    const filename = req.params.filename;
    const outputDir = process.env.OUTPUT_DIR;
    const filePath = path.join(outputDir, filename);

    fs.unlink(filePath, (err) => {
        if (err) {
            logger.error(`Failed to delete recording: ${filename}`);
            return res.status(500).json({ error: 'Failed to delete recording' });
        }
        logger.info(`Recording deleted successfully: ${filename}`);
        res.status(200).json({ message: 'Recording deleted successfully' });
    });
});

app.get("/status", (req, res) => {
  if (recordingProcess) {
    res.json({ recording: true, startTime: recordingStartTime.toISOString() });
  } else {
    res.json({ recording: false });
  }
});

app.get("/labels", (req, res) => {
  let labels = {};
  Object.keys(process.env).forEach(key => {
    if (key.startsWith("LABEL_")) {
      labels[key] = process.env[key];
    }
  });
  res.json(labels);
});

app.listen(port, () => {
  logger.info(`Server running at http://localhost:${port}`);
});