{
  apiVersion: 'argoproj.io/v1alpha1',
  kind: 'Workflow',
  metadata: {
    generateName: 'tipocket-%s-' % std.strReplace($._config.case_name, '_', '-'),
  },
  spec: {
    entrypoint: 'starter',
    onExit: 'exit-handler',
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
        outputs: {
          artifacts: [
            {
              name: 'case-logs',
              archiveLogs: true,
              path: '/logs',
            },
          ],
        },
        container: {
          name: 'tipocket',
          image: $._config.image_name,
          imagePullPolicy: 'Always',
          workingDir: '/logs',
          command: [
            'sh',
            '-c',
            $.build_command(),
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
      {
        name: 'exit-handler',
        steps: [
          [
            {
              name: 'delete-ns',
              template: 'delete-ns',
              when: '{{workflow.status}} == Succeeded',
            },
          ],
        ],
      },
    ],
  },
}
