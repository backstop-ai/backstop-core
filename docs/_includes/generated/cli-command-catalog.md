<!-- GENERATED PRODUCT TRUTH | job=cli-command-catalog | inputs=cmd/backstop | owner=/reference/#cli-command-catalog | regenerate=./scripts/generate-product-truth.sh | DO NOT EDIT -->
<!-- PRODUCT-TRUTH:BEGIN job=cli-command-catalog digest=sha256:9a0727cd1a743279847069b07d7b1eed78c034ba5bfe35dc4b238201ed6324d6 -->
<table data-product-truth-job="cli-command-catalog">
<thead><tr><th>Command</th><th>Path</th><th>Description</th><th>Flags</th></tr></thead>
<tbody>
<tr><td>artifact</td><td>artifact</td><td>Artifact lifecycle commands</td><td>--json</td></tr>
<tr><td>new</td><td>artifact new</td><td>Scaffold a new artifact</td><td>--json<br>--slug<br>--source</td></tr>
<tr><td>validate</td><td>artifact validate</td><td>Validate artifacts against schemas</td><td>--adr<br>--all<br>--bundle<br>--directive<br>--issue<br>--json<br>--plan<br>--spec</td></tr>
<tr><td>baseline</td><td>baseline</td><td>Baseline cache and artifact commands</td><td>--json</td></tr>
<tr><td>generate</td><td>baseline generate</td><td>Generate baseline JSON from full-scope gate</td><td>--json</td></tr>
<tr><td>pull</td><td>baseline pull</td><td>Fetch latest successful main baseline artifact</td><td>--json</td></tr>
<tr><td>commands</td><td>commands</td><td>List all available commands for agent discovery</td><td>--help<br>--json<br>--json</td></tr>
<tr><td>doctor</td><td>doctor</td><td>Diagnose a backstop setup, including the conditions that make other commands refuse</td><td>--check<br>--json</td></tr>
<tr><td>gate</td><td>gate</td><td>Run full verification gate</td><td>--all<br>--base<br>--file<br>--json<br>--json-out<br>--pack-sandbox</td></tr>
<tr><td>init</td><td>init</td><td>Take a project from nothing to a first gated run</td><td>--ci<br>--json<br>--no-baseline<br>--no-git<br>--no-gitignore<br>--no-observe<br>--no-packs<br>--no-sdlc<br>--no-toolchain<br>--only<br>--pack<br>--scaffold</td></tr>
<tr><td>pack</td><td>pack</td><td>Enforcement content commands</td><td>--json</td></tr>
<tr><td>add</td><td>pack add</td><td>Add an enforcement pack to the project</td><td>--json<br>--version</td></tr>
<tr><td>check</td><td>pack check</td><td>Validate pack manifest and constraints</td><td>--format<br>--json</td></tr>
<tr><td>install</td><td>pack install</td><td>Restore packs from backstop.lock</td><td>--cache<br>--json</td></tr>
<tr><td>list</td><td>pack list</td><td>List installed enforcement packs</td><td>--json</td></tr>
<tr><td>new</td><td>pack new</td><td>Scaffold a new enforcement pack</td><td>--json<br>--language<br>--slug<br>--type</td></tr>
<tr><td>relock</td><td>pack relock</td><td>Refresh a local pack&#39;s lock entry after editing it in place</td><td>--json</td></tr>
<tr><td>remove</td><td>pack remove</td><td>Remove an enforcement pack from the project</td><td>--json</td></tr>
<tr><td>test</td><td>pack test</td><td>Run full pack validation including fixture execution</td><td>--format<br>--json</td></tr>
<tr><td>update</td><td>pack update</td><td>Update a pack to the latest compatible minor/patch version</td><td>--acknowledge<br>--json</td></tr>
<tr><td>upgrade</td><td>pack upgrade</td><td>Upgrade a pack to a new major version</td><td>--json</td></tr>
<tr><td>recipe</td><td>recipe</td><td>Recipe adoption commands</td><td>--json</td></tr>
<tr><td>apply</td><td>recipe apply</td><td>Apply a pinned recipe from an installed pack</td><td>--json<br>--param</td></tr>
<tr><td>version</td><td>version</td><td>Print version and schema cohort information</td><td>--json</td></tr>
<tr><td>waiver</td><td>waiver</td><td>Inspect backstop waivers (read-only)</td><td>--json</td></tr>
<tr><td>list</td><td>waiver list</td><td>List active, expiring-soon, and unused/dangling waivers</td><td>--json</td></tr>
</tbody>
</table>
<!-- PRODUCT-TRUTH:SOURCES-BEGIN job=cli-command-catalog owner=/reference/#cli-command-catalog digest=sha256:9a0727cd1a743279847069b07d7b1eed78c034ba5bfe35dc4b238201ed6324d6 -->
<ul data-generated-source-descriptors data-product-truth-job="cli-command-catalog">
<li data-generated-source-descriptor data-source-kind="tree" data-commit-binding="site" data-source-path="cmd/backstop">https://github.com/backstop-ai/backstop-core/tree/&lt;SITE-COMMIT&gt;/cmd/backstop</li>
</ul>
<!-- PRODUCT-TRUTH:SOURCES-END job=cli-command-catalog -->
<!-- PRODUCT-TRUTH:END job=cli-command-catalog -->
