if std.extVar('build_mode') == 'workflow' then (import 'argo/workflow.libsonnet') else (import 'argo/cron_workflow.libsonnet')
