# Copyright 2026 Abhinav Gautam
# SPDX-License-Identifier: Apache-2.0
#
# Runs pyoscp's Capacity Provider endpoints with its registration
# manager, for the evroaming interop tests. Lines starting with "PEER "
# report what the peer was given.

import os

from flask import Flask
from oscp import createBlueprint
from oscp.RegistrationManager import RegistrationDictMan

HOST_URL = os.environ["HOST_URL"]
TOKEN_A = os.environ.get("PEER_TOKEN_A", "peer-token-a")


def say(*parts):
    print("PEER", *parts, flush=True)


class CapacityProvider:
    def handleAdjustGroupCapacityForecast(self, payload, token):
        say("adjust", payload.get("group_id"), len(payload.get("forecasted_blocks") or []))

    def handleGroupCapacityComplianceError(self, payload, token):
        say("compliance-error", payload.get("message"))

    def handleUpdateGroupMeasurements(self, payload, token):
        say("group-measurements", payload.get("group_id"), len(payload.get("measurements") or []))


class Nothing:
    pass


regman = RegistrationDictMan([{"version": "2.0", "base_url": HOST_URL + "/oscp/cp/2.0"}])
regman._updateService(TOKEN_A, None, None)

app = Flask(__name__)
app.register_blueprint(createBlueprint({
    "energyprovider": Nothing(), "capacityprovider": CapacityProvider(),
    "capacityoptimizer": Nothing(), "flexibilityprovider": Nothing(),
    "registrationmanager": regman}, actor="cp"))

if __name__ == "__main__":
    say("ready")
    regman.start()
    app.run(host="0.0.0.0", port=5000)
