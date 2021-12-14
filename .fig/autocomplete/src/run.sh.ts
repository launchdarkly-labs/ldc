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
}

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
    }
  ],
};
export default completionSpec;