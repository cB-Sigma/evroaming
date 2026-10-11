# Copyright 2026 Abhinav Gautam
# SPDX-License-Identifier: Apache-2.0
#
# pyoscp (MIT, github.com/NOWUM/pyoscp), an OSCP 2.0 implementation on
# Flask-RESTX, as an interop peer in the Capacity Provider role. It is
# installed from PyPI at build time, not redistributed; app.py is our own
# script that runs it. python-dateutil is a dependency pyoscp 0.1.0 uses
# but does not declare.
FROM python:3.11.9-slim
RUN pip install --no-cache-dir pyoscp==0.1.0 flask==2.3.3 werkzeug==2.3.8 flask-restx==1.3.0 requests==2.32.3 packaging==21.3 python-dateutil==2.9.0.post0
COPY app.py /peer/app.py
WORKDIR /peer
EXPOSE 5000
CMD ["python", "-u", "app.py"]
