package istio.authz

import future.keywords

import input.attributes.request.http as http_request
import input.parsed_path

default allow := false

allow if {
	parsed_path[0] == "health"
	http_request.method == "GET"
}

allow if {
	some r in roles_for_user
	r in required_roles
}

roles_for_user contains r if {
	some r in user_roles[user_name]
}

required_roles contains r if {
	some perm in role_perms[r]
	perm.method == http_request.method
	perm.path == http_request.path
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