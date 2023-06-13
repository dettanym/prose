   package istio.authz

    import input.attributes.request.http as http_request
    import input.parsed_path

    default allow = false

    allow {
        parsed_path[0] == "health"
        http_request.method == "GET"
    }

    allow {
        roles_for_user[r]
        required_roles[r]
    }

    roles_for_user[r] {
        r := user_roles[user_name][_]
    }

    required_roles[r] {
        perm := role_perms[r][_]
        perm.method = http_request.method
        perm.path = http_request.path
    }

    user_name := http_request.headers.authorization

    user_roles = {
        "alice": ["guest"],
        "bob": ["admin"]
    }

    role_perms = {
        "guest": [
            {"method": "GET",  "path": "/productpage"},
        ],
        "admin": [
            {"method": "GET",  "path": "/productpage"},
            {"method": "GET",  "path": "/api/v1/products"},
        ],
    }