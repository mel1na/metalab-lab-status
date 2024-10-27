# metalab-lab-status
This go application scrapes the Metalab Home Assistant REST API and returns information about the `lab-is-on` toggle.

## Requirements
- .env file with a long-lived `HOMEASSISTANT_TOKEN` set
- Connection to Home Assistant

## Compiling
For Raspberry Pi (3B+): `env GOOS=linux GOARCH=arm64 go build`