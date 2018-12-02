// Copyright 2018 SpotHero
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

'use strict';

const path = require('path');
const Generator = require('yeoman-generator');
const mkdir = require('mkdirp');

module.exports = class extends Generator {

    paths() {
        this.destinationRoot(process.env.GOPATH || './');
    }

    prompting() {
        let cb = this.async();

        let prompts = [{
            type: 'input',
            name: 'appName',
            message: 'What is the name of your application?',
            default: 'helloworld'
        }, {
            type: 'input',
            name: 'repoUrl',
            message: 'Where will the repository be located under GOPATH?',
            default: 'github.com/spothero'
        }];

        return this.prompt(prompts).then(props => {
            this.appName = props.appName.replace(/\s+/g, '-').toLowerCase();
            if (props.repoUrl.substr(-1) != '/') props.repoUrl += '/';
            this.repoUrl = props.repoUrl + this.appName;
            cb()
        });

    }

    writing() {
        let tmplContext = {
            appName: this.appName,
            repoUrl: this.repoUrl
        };

        console.log('Generating tree folders');
        let cmdDir = this.destinationPath('cmd');
        let pkgDir = this.destinationPath('pkg');
        let srcDir = this.destinationPath(path.join('src/', this.repoUrl));
        let binDir = this.destinationPath('bin');

        mkdir.sync(cmdDir)
        mkdir.sync(pkgDir);
        mkdir.sync(srcDir);
        mkdir.sync(binDir);

        this.fs.copyTpl(
            this.templatePath('_gitignore'),
            path.join(srcDir, '.gitignore'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_Makefile'),
            path.join(srcDir, '/Makefile'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_Gopkg.toml'),
            path.join(srcDir, '/Gopkg.toml'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_Gopkg.lock'),
            path.join(srcDir, '/Gopkg.lock'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_Dockerfile'),
            path.join(srcDir, '/Dockerfile'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_docker-compose.yaml'),
            path.join(srcDir, '/docker-compose.yaml'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_README.md'),
            path.join(srcDir, 'README.md'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_cmd.go'),
            path.join(srcDir, 'cmd/cmd.go'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_routes.go'),
            path.join(srcDir, 'pkg/' + this.appName + '/routes.go'),
            tmplContext
        );
        this.fs.copyTpl(
            this.templatePath('_routes_test.go'),
            path.join(srcDir, 'pkg/' + this.appName + '/routes_test.go'),
            tmplContext
        );
    }
};
