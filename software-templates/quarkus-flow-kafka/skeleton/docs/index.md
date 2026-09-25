# ${{values.component_id}}

Agentic Trip Planner ([quarkus-workshop-langchain4j](https://quarkus.io/quarkus-workshop-langchain4j/) section-3/step-04): **Red Hat langchain4j** agents + Kafka for human-in-the-loop approval after planning. Application CI/CD runs on **OpenShift Pipelines**. GitLab CI only builds TechDocs.

## Demo path

1. Open Dev Spaces from the component links.
2. Ensure `MAAS_API_KEY` is set (`.env` task).
3. Run Quarkus dev (JDK 21).
4. Plan a trip from the UI.
5. When the agent pauses, use **Approve** / **Reject** (Kafka `flow-in` / `flow-out`).
