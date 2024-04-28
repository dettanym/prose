from importlib import import_module
from os import chdir
from sys import argv, exit

if __name__ == "__main__":
    args = argv[1:]

    if len(args) == 0:
        print("something is not right")
        exit(1)

    app = args[0]
    if app == "#!":
        app = args[1]
        real_cwd = args[2]
        arguments = args[3:]

        chdir(real_cwd)
    else:
        arguments = args[1:]

    if app.endswith(".py"):
        app = app.removesuffix(".py")

    if app.startswith(__package__ + "/"):
        app = app.removeprefix(__package__ + "/")

    if not app.startswith("."):
        app = "." + app

    imported = import_module(app, package=__package__)
    imported.main(*arguments)
