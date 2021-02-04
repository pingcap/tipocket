if std.extVar('build_mode') == 'workflow' then (import 'argo/workflow.jsonnet') else (import 'argo/cron_workflow.jsonnet')
