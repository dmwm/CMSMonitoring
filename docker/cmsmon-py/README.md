# CMS Monitoring Python base image

Dockerfile for generating the Base image for most non-Spark Python services in CMS Monitoring infrastructure.

## Characteristics

### Base Image

- Python 3.9 slim base image from Docker Hub via CERN pull-through cache
- Working directory: `/data`
- Environment variables:
  - `PYTHONPATH` includes `/data`
  - `LC_ALL=C.UTF-8` and `LANG=C.UTF-8` for proper Unicode support

### Pre-installed Packages

- `click` - Command line interface creation toolkit
- `pandas` - Data manipulation and analysis library
- `rucio-clients` - CMS data management system client
- `schema` - Data validation library
- `stomp.py==7.0.0` - STOMP protocol implementation for message queuing

## Usage

This image functions as a base image for CMS Monitoring services that require Python but are not Spark-related. It is designed to be extended rather than run directly, and should be used as a base layer in Docker images that contain the specific scripts and code for individual services.
