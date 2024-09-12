FROM python:3.10
RUN pip install numpy matplotlib
COPY . /data
WORKDIR /data
CMD [ "python3", "/data/analysis.py" ]
