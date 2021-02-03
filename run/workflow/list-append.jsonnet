(import "argo.jsonnet") +
(import "case.jsonnet") +
{
    _config+:: {
        case_name: "list-append",
        image_name: "hub.pingcap.net/mahjonp/tipocket",
        command: std.join(" \\\n", $.list_append(tablecount="7", read_lock="\"FOR UPDATE\"", txn_mode="pessimistic") + $.fixture(namespace="{{workflow.name}}", delete_ns="true", storage_class="local-storage"))
    },
}