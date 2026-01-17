const express = require("express");
const ffmpeg = require("fluent-ffmpeg");
const path = require("path");
const fs = require("fs");
const dotenv = require("dotenv");
const winston = require("winston");
const { log } = require("console");

// Configure dotenv
// When running as pkg executable, use the directory where the .exe is located
// When running as Node.js script, use __dirname
const isPackaged = typeof process.pkg !== 'undefined';
const baseDir = isPackaged ? path.dirname(process.execPath) : __dirname;
const envFile = process.env.ENV_FILE || 'prod.env';
// If ENV_FILE is an absolute path, use it directly; otherwise join with baseDir
const envPath = path.isAbsolute(envFile) ? envFile : path.join(baseDir, envFile);

dotenv.config({ path: envPath });

// Public folder path - for pkg, assets are in snapshot (__dirname)
// For env file and logs, use baseDir (where .exe is located)
const publicDir = isPackaged ? path.join(__dirname, 'public') : path.join(__dirname, 'public');

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
        new winston.transports.File({ filename: path.join(baseDir, 'server.log') }),
        new winston.transports.Console()
    ]
});

// Log configuration for debugging
logger.info(`Running as: ${isPackaged ? 'Packaged executable' : 'Node.js script'}`);
logger.info(`Base directory: ${baseDir}`);
logger.info(`Public directory: ${publicDir}`);
logger.info(`Environment file: ${envPath} (exists: ${fs.existsSync(envPath)})`);

const app = express();
const port = process.env.PORT || 8080;

let recordingProcess = null;
let recordingStartTime = null;
let currentRecordingFile = null;

// Helper function to build RTSP URL
function buildRtspUrl() {
  const { RTSP_USER, RTSP_PASSWORD, RTSP_IP, RTSP_PORT, RTSP_SDP } = process.env;
  
  if (RTSP_USER && RTSP_PASSWORD) {
    return `rtsp://${RTSP_USER}:${RTSP_PASSWORD}@${RTSP_IP}:${RTSP_PORT}/${RTSP_SDP}`;
  } else {
    return `rtsp://${RTSP_IP}:${RTSP_PORT}/${RTSP_SDP}`;
  }
}

app.use(express.static(publicDir));

app.get("/start", (req, res) => {
  if (recordingProcess) {
    logger.warn("Attempt to start recording while another is in progress");
    return res.status(400).send("Recording already in progress");
  }

  const rtspUrl = buildRtspUrl();
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
  currentRecordingFile = `${timestamp}.mp4`;

  let responseSent = false;
  let connectionVerified = false;

  // Start recording if the stream is available
  recordingProcess = ffmpeg(rtspUrl)
    .inputOptions(["-rtsp_transport tcp"])
    .outputOptions(["-c:v copy", "-c:a copy", "-movflags +faststart"])
    .save(outputPath)
    .on("start", (output) => {
      logger.info("FFmpeg process started");
      logger.info(output);
    })
    .on("codecData", (data) => {
      // This event fires when ffmpeg successfully reads stream info
      if (!connectionVerified) {
        connectionVerified = true;
        recordingStartTime = now;
        logger.info("Recording started - connection verified");
        if (!responseSent) {
          responseSent = true;
          res.json({
            message: "Recording started",
            startTime: recordingStartTime.toISOString(),
          });
        }
      }
    })
    .on("end", (output) => {
      logger.info("Recording finished");
      logger.info(output);
      recordingProcess = null;
      recordingStartTime = null;
      currentRecordingFile = null;
    })
    .on("error", (err) => {
      logger.error(`FFmpeg error: ${err.message}`);
      recordingProcess = null;
      recordingStartTime = null;
      currentRecordingFile = null;
      
      if (!responseSent) {
        responseSent = true;
        res.status(500).json({ 
          error: "Failed to connect to camera",
          message: err.message 
        });
      }
    });

  // Timeout after 30 seconds if connection is not verified
  setTimeout(() => {
    if (!connectionVerified && !responseSent) {
      responseSent = true;
      logger.error("Connection timeout - unable to connect to camera");
      if (recordingProcess) {
        recordingProcess.kill("SIGKILL");
        recordingProcess = null;
        recordingStartTime = null;
        currentRecordingFile = null;
      }
      res.status(500).json({ 
        error: "Connection timeout",
        message: "Unable to connect to camera within 30 seconds" 
      });
    }
  }, 30000);
});

app.get("/stop", (req, res) => {
  if (!recordingProcess) {
    logger.warn("Attempt to stop recording when no recording is in progress");
    return res.status(400).send("No recording in progress");
  }

  recordingProcess.kill("SIGINT");
  recordingProcess = null;
  recordingStartTime = null; // Reset the start time
  currentRecordingFile = null; // Clear the current recording file
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
      .filter((file) => file !== currentRecordingFile) // Exclude current recording
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