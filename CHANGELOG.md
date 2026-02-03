# Changelog

All notable changes to this project will be documented in this file. See [commit-and-tag-version](https://github.com/absolute-version/commit-and-tag-version) for commit guidelines.

## [1.3.6](https://github.com/joaoopereira/simple-rtsp-recorder/compare/v1.3.5...v1.3.6) (2026-02-03)

## [1.3.5](https://github.com/joaoopereira/simple-rtsp-recorder/compare/v1.3.4...v1.3.5) (2026-02-03)

## [1.3.4](https://github.com/joaoopereira/simple-rtsp-recorder/compare/v1.3.3...v1.3.4) (2026-01-20)


### Bug Fixes

* handle graceful stopping of recording and improve error logging ([986e901](https://github.com/joaoopereira/simple-rtsp-recorder/commit/986e90137e75c806a52013873b4729a460665e43))
* improve installer script and README ([a142fdd](https://github.com/joaoopereira/simple-rtsp-recorder/commit/a142fddff6f5b3d6057d1c1e092787c0762764fb))
* recording not stopping - clear state immediately with delayed flag reset ([57ff412](https://github.com/joaoopereira/simple-rtsp-recorder/commit/57ff4120acceee157a36dc69115935ecfbc48019))

## [1.3.3](https://github.com/joaoopereira/simple-rtsp-recorder/compare/v1.3.2...v1.3.3) (2026-01-20)


### Bug Fixes

* race condition allowing multiple ffmpeg processes to run simultaneously ([70e72be](https://github.com/joaoopereira/simple-rtsp-recorder/commit/70e72bed7ab80f859ab55aa8f3ffe78dcaf9748c))

## [1.3.2](https://github.com/joaoopereira/simple-rtsp-recorder/compare/v1.3.1...v1.3.2) (2026-01-17)


### Bug Fixes

* remove GITHUB_TOKEN from release workflow and improve uninstall process in installer script ([069eeb8](https://github.com/joaoopereira/simple-rtsp-recorder/commit/069eeb8661e2da832a1398b513ccece5b92f484a))

## [1.3.1](https://github.com/joaoopereira/simple-rtsp-recorder/compare/v1.3.0...v1.3.1) (2026-01-17)


### Bug Fixes

* update bump script to include git push with tags ([346fca5](https://github.com/joaoopereira/simple-rtsp-recorder/commit/346fca59225725c8bbda1a5058a8bf0934d4a5cd))

## [1.3.0](https://github.com/joaoopereira/simple-rtsp-recorder/compare/v1.2.0...v1.3.0) (2026-01-17)


### Features

* add Windows installer script and GitHub Actions workflow for automated releases ([9ef8816](https://github.com/joaoopereira/simple-rtsp-recorder/commit/9ef88169ce5125a8f4f478964b1bb2daf125326b))

## 1.2.0 (2026-01-17)


### Features

* add Windows service installer and uninstaller scripts, along with service configuration ([b9655e5](https://github.com/joaoopereira/simple-rtsp-recorder/commit/b9655e557f5f756d624d96d8fc5e5d31a07971ce))
* enhance error handling for recording start and connection verification ([31233be](https://github.com/joaoopereira/simple-rtsp-recorder/commit/31233be68ca43f18c247155e52468bbef3189835))
* implement helper function to build RTSP URL and update recording logic ([b2150eb](https://github.com/joaoopereira/simple-rtsp-recorder/commit/b2150eb5cd2dcd9c0fd77246e2d76b59b787b23e))
* improve dotenv configuration and logging for better environment handling ([3c83d0a](https://github.com/joaoopereira/simple-rtsp-recorder/commit/3c83d0abe8218eb535c69d27d16eab5a4674e82a))
* initial version ([e9a0997](https://github.com/joaoopereira/simple-rtsp-recorder/commit/e9a099706ce71c0d1cfb8927260affc35d02eb2f))

## 1.1.0 (2026-01-17)


### Features

* enhance error handling for recording start and connection verification ([31233be](https://github.com/joaoopereira/simple-rtsp-recorder/commit/31233be68ca43f18c247155e52468bbef3189835))
* implement helper function to build RTSP URL and update recording logic ([b2150eb](https://github.com/joaoopereira/simple-rtsp-recorder/commit/b2150eb5cd2dcd9c0fd77246e2d76b59b787b23e))

## 1.0.0 (2024-09-10)


### Features

* initial version ([e9a0997](https://github.com/joaoopereira/simple-rtsp-recorder/commit/e9a099706ce71c0d1cfb8927260affc35d02eb2f))
