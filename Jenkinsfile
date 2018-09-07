/// Build Pipeline Code
node('docker') {
    stage('Prep Environment') {
        def scmVars = checkout([$class : 'GitSCM', branches: [[name: env.BRANCH_NAME]], doGenerateSubmoduleConfigurations: false, extensions: [], gitTool: 'Default', submoduleCfg: [],
                  userRemoteConfigs: [[credentialsId: '7ba26c51-4710-4d39-b610-68ab11203c11', url: 'https://github.com/spothero/availability.git']]])
        env.GIT_COMMIT = scmVars.GIT_COMMIT
        // Nexus rejects branch names including `/`, clean them from the branch name
        env.BRANCH_NAME = env.BRANCH_NAME.replaceAll('/', '-')
    }
    stage(Test') {
        sh """
           echo "######################### Building Docker Container and Test #############################"
           docker run -it -v `pwd`:/go/src/github.com/spothero/core -w /go/src/github.com/spothero/core golang:latest make test
           echo "######################### Completing Docker Container Build and Test #############################"
           """
    }
}
