function envoy_on_request(request_handle)
    request_handle:logWarn("hello world >> on request")
    -- Log information about the request
    request_handle:logWarn("  Authority: "..request_handle:headers():get(":authority"))
    request_handle:logWarn("  Method: "..request_handle:headers():get(":method"))
    request_handle:logWarn("  Path: "..request_handle:headers():get(":path"))
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
    -- Make a call to the presidio service at presidio.prose-system.svc.cluster.local
    local headers, body = response_handle:httpCall(
            "prose_cluster",
            {
                [":method"] = "GET",
                -- Create an HTTP body with a json_to_analyze field
                [":path"] = "/batchanalyze",
                [":authority"] = "presidio.prose-system.svc.cluster.local:3000",
                ["Content-Type"] = "application/json"
            }
    ,
            '{"json_to_analyze": { "URL": "www.abc.com"}}'
    )
    for key, value in pairs(headers) do
        response_handle:logWarn("Presidio response header key: "..key..", Value: "..value)
    end
    for key, value in pairs(body) do
        response_handle:logWarn("Presidio response body key: "..key..", Value: "..value)
    end
end