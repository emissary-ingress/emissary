FROM python:3.7-slim

RUN pip install flask

COPY server.py .
COPY openapi.json .

ENV FLASK_APP=server.py
CMD ["flask", "run", "-h", "0.0.0.0", "-p", "5000"]
