// Brand colors
const LD_BLUE_HEX = '405BFF';
const LD_CYAN_HEX = '3DD6F5';
const LD_PURPLE_HEX = 'A34FDE';

const configNameGenerator: Fig.Generator = {
  //TODO this should optionally use a config specified with --config
  script: "jq 'keys' ~/.config/ldc.json",
  postProcess: (out) => {
    const configs: Array<string> = JSON.parse(out);

    return configs.map<Fig.Suggestion>((item) => {
      return {
        name: item,
        insertValue: item,
        description: item,
      };
    });
  }
};

const projectGenerator: Fig.Generator = {
  script(context) {
    let cmd = './run.sh projects list';
    const config = getOptionFromContext(context, configOpt);
    if (config) {
      cmd += ` --config ${config}`;
    }
    const configFile = getOptionFromContext(context, configFileOpt);
    if (configFile) {
      cmd += ` --config-file ${configFile}`;
    }

    return cmd;
  },
  postProcess(out) {
    return out.split("\n").reduce((acc, line) => {
      const match = line.match(
        /^\| (?<key>[^\s]+) +\| (?<name>.+\b)\s+\|$/
      );
  
      if (match !== null) {
        const { key, name } = match.groups;
        acc.push({
          name: key,
          insertValue: key,
          description: name,
          icon: `fig://template?color=${LD_BLUE_HEX}&badge=P`
        });
      }

      return acc;
    }, []);
  },
};

const environmentGenerator: Fig.Generator = {
  script(context) {
    let cmd = './run.sh environments list';
    const config = getOptionFromContext(context, configOpt);
    if (config) {
      cmd += ` --config ${config}`;
    }
    const configFile = getOptionFromContext(context, configFileOpt);
    if (configFile) {
      cmd += ` --config-file ${configFile}`;
    }

    return cmd;
  },
  postProcess(out) {
    return out.split("\n").reduce((acc, line) => {
      const match = line.match(
        /^\| (?<key>[^\s]+) +\| (?<name>.+\b)\s+\|$/
      );
  
      if (match !== null) {
        const { key, name } = match.groups;
        acc.push({
          name: key,
          insertValue: key,
          description: name,
          icon: `fig://template?color=${LD_CYAN_HEX}&badge=E`
        });
      }

      return acc;
    }, []);
  },
};

const flagGenerator: Fig.Generator = {
  script(context) {
    let cmd = './run.sh flags list';
    const config = getOptionFromContext(context, configOpt);
    if (config) {
      cmd += ` --config ${config}`;
    }
    const configFile = getOptionFromContext(context, configFileOpt);
    if (configFile) {
      cmd += ` --config-file ${configFile}`;
    }

    return cmd;
  },
  postProcess(out) {
    return out.split("\n").reduce((acc, line) => {
      const match = line.match(
        /^\| (?<key>[^\s]+) +\| (?<name>.+\b)\s+\| (?<partialDescription>.+)\|/
      );
  
      if (match !== null) {
        const { key, name } = match.groups;
        acc.push({
          name: key,
          insertValue: key,
          description: name,
          icon: `fig://template?color=${LD_PURPLE_HEX}&badge=⚑`
        });
      }

      return acc;
    }, []);
  },
};

const getOptionFromContext = (context, option: Fig.Option) => {
  const index = getOptionIndexFromContext(context, option);
  const value = index > -1 ? context[index+1] : '';

  return value;
}

const getOptionIndexFromContext = (context, option: Fig.Option) => {
  for (const name of option.name) {
    const idx = context.indexOf(name);
    if (idx > -1) {
      return idx;
    }
  }

  return -1;
}

const configOpt: Fig.Option = {
  name: ["--config"],
  description: "Configuration to use",
  args: [
    {
      name: "config name",
      generators: configNameGenerator,
    },
  ],
};

const configFileOpt: Fig.Option = {
  name: ["--config-file"],
  description: "Configuration file to use",
  args: [
    {
      name: "config file path",
      template: "filepaths",
    },
  ],
};

const completionSpec: Fig.Spec = {
  name: "ldc",
  description: "ldc is a command-line api client for LaunchDarkly",
  subcommands: [
    {
      name: "configs",
      description: "Manage configurations",
      subcommands: [
        {
          name: "add",
          description: "add config <config name> <api token> <project> <environment> [server url]",
          args: [
            {
              name: "config name",
            },
            {
              name: "api token",
            },
            {
              name: "project",
              description: "default project key",
            },
            {
              name: "environment",
              description: "default environment key",
            },
            {
              name: "server url",
              isOptional: true,
            },
          ],
        },
        {
          name: "set",
          description: "Change configuration <config name>",
          args: [
            {
              name: "config name",
              generators: configNameGenerator,
            },
          ],
        },
        {
          name: ["rename", "rn", "mv"],
          description: "Rename config <config name> <new name>",
          args: [
            {
              name: "config name",
              generators: configNameGenerator,
            },
            {
              name: "new name",
            },
          ],
        },
        {
          name: ["edit", "update"],
          description: "Update config: <config name> <api token> <project> <environment> [server url]",
          args: [
            {
              name: "config name",
              generators: configNameGenerator,
            },
            {
              name: "api token",
            },
            {
              name: "project",
              description: "default project key",
            },
            {
              name: "environment",
              description: "default environment key",
            },
            {
              name: "server url",
              isOptional: true,
            },
          ],
        },
        {
          name: ["rm", "remove", "delete", "del"],
          description: "Remove config: <config name>",
          args: [
            {
              name: "config name",
              generators: configNameGenerator,
            },
          ],
        },
      ],
    },
    {
      name: "projects",
      description: "Create, list, view, and delete projects",
      subcommands: [
        {
          name: "list",
          description: "List projects",
        },
        {
          name: "show",
          description: "View project details",
          args: [
            {
              name: "project key",
              generators: projectGenerator,
              debounce: true,
            },
          ],
        },
        {
          name: ["create", "new"],
          description: "Create a project",
          args: [
            {
              name: "project key",
            },
          ],
        },
        {
          name: ["delete", "remove"],
          description: "Delete a project",
          args: [
            {
              name: "project key",
              isDangerous: true,
              generators: projectGenerator,
              debounce: true,
            },
          ],
        },
      ],
    },
    {
      name: ["environments", "environment", "envs", "env", "e"],
      description: "Create, list, view, and delete environments",
      subcommands: [
        {
          name: ["list", "ls", "l"],
          description: "List environments",
        },
        {
          name: "show",
          description: "View environments details",
          args: [
            {
              name: "environments key",
              generators: environmentGenerator,
              debounce: true,
            },
          ],
        },
        {
          name: ["create", "new", "c"],
          description: "Create an environment",
          args: [
            {
              name: "project key",
            },
          ],
        },
        {
          name: ["delete", "remove", "d", "del", "rm"],
          description: "Delete an environment",
          args: [
            {
              name: "envitonment key",
              isDangerous: true,
              generators: environmentGenerator,
              debounce: true,
            },
          ],
        },
      ],
    },
    {
      name: ["flags", "flag"],
      description: "Create, list, view, and delete flags",
      subcommands: [
        {
          name: ["list", "ls", "l"],
          description: "List flags",
        },
        {
          name: "show",
          description: "View flag details",
          args: [
            {
              name: "flag key",
              generators: flagGenerator,
              debounce: true,
            },
          ],
        },
        {
          name: ["create", "new"],
          description: "Create a flag",
          args: [
            {
              name: "flag key",
            },
          ],
        },
        {
          name: ["create-toggle", "new-toggle", "create-boolean"],
          description: "Create a boolean flag",
          args: [
            {
              name: "flag key",
            },
          ],
        },
        {
          name: "add-tag",
          description: "Add a tag to a flag",
          args: [
            {
              name: "flag key",
              generators: flagGenerator,
              debounce: true,
            },
            {
              name: "tag",
            },
          ],
        },
        {
          name: "remove-tag",
          description: "Remove a tag from a flag",
          args: [
            {
              name: "flag key",
              generators: flagGenerator,
              debounce: true,
            },
            {
              name: "tag",
            },
          ],
        },
        {
          name: "status",
          description: "Show flag's statuses",
          args: [
            {
              name: "flag key",
              generators: flagGenerator,
              debounce: true,
            },
          ],
        },
        {
          name: "on",
          description: "Turn a boolean flag on",
          args: [
            {
              name: "flag key",
              generators: flagGenerator,
              debounce: true,
            },
          ],
        },
        {
          name: "off",
          description: "Turn a boolean flag off",
          args: [
            {
              name: "flag key",
              generators: flagGenerator,
              debounce: true,
            },
          ],
        },
        {
          name: ["delete", "remove"],
          description: "Delete a flag",
          args: [
            {
              name: "flag key",
              isDangerous: true,
              generators: flagGenerator,
              debounce: true,
            },
          ],
        },
      ],
    },
  ],
  options: [
    configOpt,
    configFileOpt,
  ],
};
export default completionSpec;