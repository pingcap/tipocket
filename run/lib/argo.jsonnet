{
  apiVersion: 'argoproj.io/v1alpha1',
  kind: 'Workflow',
  metadata: {
    generateName: 'tipocket-%s-' % $._config.case_name,
  },
  spec: {
    entrypoint: 'starter',
    onExit: 'delete-ns',
    templates: [
      {
        name: 'starter',
        steps: [
          [
            {
              name: 'create-ns',
              template: 'create-ns',
            },
          ],
          [
            {
              name: 'run-tipocket',
              template: 'run-tipocket',
            },
          ],
        ],
      },
      {
        name: 'run-tipocket',
        container: {
          name: 'tipocket',
          image: $._config.image_name,
          imagePullPolicy: 'Always',
          command: [
            'sh',
            '-c',
            std.join(' \\n', $._config.command),
          ],
        },
      },
      {
        name: 'create-ns',
        resource: {
          action: 'create',
          successCondition: 'status.phase = Active',
          manifest: 'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: {{workflow.name}}\n',
        },
      },
      {
        name: 'delete-ns',
        resource: {
          action: 'delete',
          manifest: 'apiVersion: v1\nkind: Namespace\nmetadata:\n  name: {{workflow.name}}\n',
        },
      },
    ],
  },
}
