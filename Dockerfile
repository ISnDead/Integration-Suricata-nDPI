FROM ubuntu:latest

RUN apt-get update && apt-get install -y \
    sudo \
    curl

USER root
