import json
import time
from flask import Flask, make_response
app = Flask(__name__)

@app.route('/.ambassador-internal/openapi-docs')
def openapi_docs():
    with open("openapi.json") as f:
        docs = json.load(f)
    docs["info"]["title"] += " generated @ " + time.asctime()
    result = make_response(json.dumps(docs))
    result.headers["content-type"] = "text/json"
    return result
