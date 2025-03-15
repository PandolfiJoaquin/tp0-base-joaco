FROM ubuntu:22.04
COPY . .

RUN apt update && apt install netcat -y

RUN chmod +x netcat_script.sh
ENTRYPOINT ["/bin/bash", "-c", "./netcat_script.sh"]



