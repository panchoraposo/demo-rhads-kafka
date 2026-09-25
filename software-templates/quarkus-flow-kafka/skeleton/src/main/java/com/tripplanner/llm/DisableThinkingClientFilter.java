package com.tripplanner.llm;

import java.util.LinkedHashMap;
import java.util.Map;

import org.eclipse.microprofile.config.inject.ConfigProperty;
import org.jboss.resteasy.reactive.client.spi.ResteasyReactiveClientRequestContext;
import org.jboss.resteasy.reactive.client.spi.ResteasyReactiveClientRequestFilter;

import dev.langchain4j.model.openai.internal.chat.ChatCompletionRequest;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.ws.rs.ext.Provider;

/**
 * Qwen3 thinks by default ({@code reasoning_content}), which makes every MaaS
 * call 5–17s. LiteLLM/vLLM honor {@code chat_template_kwargs.enable_thinking=false}.
 */
@Provider
@ApplicationScoped
public class DisableThinkingClientFilter implements ResteasyReactiveClientRequestFilter {

    @ConfigProperty(name = "demo.llm.disable-thinking", defaultValue = "false")
    boolean disableThinking;

    @Override
    public void filter(ResteasyReactiveClientRequestContext requestContext) {
        if (!disableThinking) {
            return;
        }
        Object entity = requestContext.getEntity();
        if (!(entity instanceof ChatCompletionRequest request)) {
            return;
        }
        Map<String, Object> extra = new LinkedHashMap<>();
        if (request.customParameters() != null) {
            extra.putAll(request.customParameters());
        }
        extra.put("chat_template_kwargs", Map.of("enable_thinking", false));
        requestContext.setEntity(
                ChatCompletionRequest.builder().from(request).customParameters(extra).build());
    }
}
