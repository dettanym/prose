function envoy_on_request(request_handle)
    -- Make an HTTP call to an upstream host with the following headers, body, and timeout.
    -- local headers, body = request_handle:httpCall(
    --  "lua_cluster",
    --  {
    --   [":method"] = "POST",
    --   [":path"] = "/acl",
    --   [":authority"] = "internal.org.net"
    --  },
    -- "authorize call",
    -- 5000)

    request_handle:logWarn("hello world >> on request")
    -- Log information about the request
    request_handle:logWarn("  Authority: "..request_handle:headers():get(":authority"))
    request_handle:logWarn("  Method: "..request_handle:headers():get(":method"))
    request_handle:logWarn("  Path: "..request_handle:headers():get(":path"))
    local body_len = request_handle:body():length()
    request_handle:logWarn("  Body: "..request_handle:body():getBytes(0, body_len))
    local streamInfo = request_handle:streamInfo()
    request_handle:logWarn("Request handle downstream local address: "..streamInfo:downstreamLocalAddress()..", remote addr:"..streamInfo:downstreamDirectRemoteAddress())

end

function envoy_on_response(response_handle)
    response_handle:logWarn("hello world >> on response")
    -- Log response status code
    response_handle:logWarn("  Response handle status: "..response_handle:headers():get(":status"))
    local streamInfo = response_handle:streamInfo()
    response_handle:logWarn("Response handle downstream local address: "..streamInfo:downstreamLocalAddress()..", remote addr:"..streamInfo:downstreamDirectRemoteAddress())
    local body_len = response_handle:body():length()
    response_handle:logWarn("  Body: "..response_handle:body():getBytes(0, body_len))
    response_handle:logWarn("Request handle downstream local address: "..streamInfo:downstreamLocalAddress()..", remote addr:"..streamInfo:downstreamDirectRemoteAddress())
end
