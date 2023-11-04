FROM python:3.9

ADD requirements.txt /requirements.txt
ADD rolldice-server.py /rolldice-server.py

RUN pip install -r /requirements.txt

RUN opentelemetry-bootstrap -a install

ENTRYPOINT ["opentelemetry-instrument", "python", "/rolldice-server.py"]