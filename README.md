# fase3-apps

## Tags das imagens no ECR

Os cinco pipelines publicam imagens com o formato `<versão>-<sha de 7 caracteres>`,
por exemplo `v1.0.0-a1b2c3d`. A versão vem da tag Git `vMAJOR.MINOR.PATCH`
mais próxima no histórico do commit; enquanto não houver tags, usam `v1.0.0`.

Pushes na `main` publicam os serviços selecionados pelos filtros de caminhos.
Pushes de tags `v*` executam os cinco pipelines e publicam as imagens da versão,
após as verificações existentes. Pull requests apenas constroem e validam.
Tags de versão fora do formato `vMAJOR.MINOR.PATCH` interrompem a publicação.

As imagens já existentes no ECR mantêm suas tags. O novo padrão passa a valer
quando os workflows atualizados forem executados no GitHub.
