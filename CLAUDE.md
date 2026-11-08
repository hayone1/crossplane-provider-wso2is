# crossplane-provider-wso2is

Crossplane provider for WSO2 Identity Server, generated with Upjet from the Terraform provider
`hayone1/wso2is` (local copy: `../terraform-provider-wso2is`).

## Every session

1. **Start:** read [HANDOFF.md](HANDOFF.md), then [BRIEF.md](BRIEF.md) if you have not read it
   in this session. Use the `session-handoff` skill.
2. **End:** update HANDOFF.md before you stop, even when the work is unfinished.

## Rules

- Ask before outward-facing actions: creating or pushing repositories, publishing images or
  packages, creating or destroying cloud servers, and changing account settings.
- Follow `../crossplane-provider-hostinger`, the user's finished Upjet provider, for structure,
  CI, release pipeline and docs. The main difference: WSO2's Terraform provider uses the plugin
  framework (`WithTerraformPluginFrameworkProvider`), not the plugin SDK v2.
- Keep the template's `make`-based build. Scratch files go in the git-ignored `_debug/` folder.
- Docs: clear, tutorial-like, ninth-grade English in complete sentences
  (`contributing/docs-style.md`, `write-docs` skill).
- Test against a real WSO2 IS 7.3 (the lab in BRIEF.md section 6). Something is done only when it
  works against the server.
- Fix Terraform provider bugs upstream in `../terraform-provider-wso2is`, with a test, and follow
  its `CLAUDE.md` there. Document every bug that you cannot fix.
- Never print, commit or write secret values into HANDOFF.md. Credentials are in `.env`.

## References

- Upjet docs: https://github.com/crossplane/upjet/tree/main/docs
- WSO2 IS docs: https://is.docs.wso2.com/en/latest/ (source: https://github.com/wso2/docs-is)
- WSO2 IS source: https://github.com/wso2/product-is
