from random import randint
from flask import Flask

app = Flask(__name__)


@app.route("/rolldice")
def roll_dice():
    return str(roll())


def roll():
    return randint(1, 6)


app.run(host="0.0.0.0")
