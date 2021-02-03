{
  build_command()::
    std.join(' \\\n', $._config.command + ['-%s=%s' % [k, $._config.args[k]] for k in std.objectFields($._config.args)]),
}
