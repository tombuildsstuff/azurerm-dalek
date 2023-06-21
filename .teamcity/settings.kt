package _Self.buildTypes

import jetbrains.buildServer.configs.kotlin.v2018_2.*
import jetbrains.buildServer.configs.kotlin.v2018_2.buildSteps.script
import jetbrains.buildServer.configs.kotlin.v2019_2.vcs.GitVcsRoot

object Dalek : Template({
    name = "Dalek"
    params {
        param("env.YES_I_REALLY_WANT_TO_DELETE_THINGS", "true")
    }
    vcs {
        root(dalekRepository)
    }

    steps {
        script {
            name = "Build"
            id = "BUILD_DALEK"
            scriptContent = """
                go mod vendor
                go build .
            """.trimIndent()
            formatStderrAsError = true
        }
        script {
            name = "Run the Dalek"
            id = "RUN_DALEK"
            scriptContent = "./azurerm-dalek %ARGS%"
            formatStderrAsError = true
        }
    }
})

object dalekRepository : GitVcsRoot({
    name = "azurerm-dalek"
    url = "https://github.com/tombuildsstuff/azurerm-dalek.git"
    agentCleanPolicy = AgentCleanPolicy.ALWAYS
    agentCleanFilesPolicy = AgentCleanFilesPolicy.ALL_UNTRACKED
    branchSpec = "+:*"
    branch = "refs/heads/main"
    authMethod = anonymous()
})
