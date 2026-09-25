package com.tripplanner.devui;

import io.vertx.core.http.HttpMethod;
import io.vertx.ext.web.Router;
import io.vertx.ext.web.RoutingContext;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.enterprise.event.Observes;

/**
 * Quarkus {@code quarkus.dev-ui.context-root} prefixes import maps and most Dev UI
 * scripts, but still emits absolute {@code /q/_static/...} for {@code lumo.css} and
 * {@code es-module-shims.js}. Behind Dev Spaces path proxy those two 404 at the
 * cluster root and the UI freezes on the blue splash — rewrite them when the
 * context root env is set by the Dev Spaces task.
 */
@ApplicationScoped
public class DevUiContextRootFix {

    private static final String SKIP = "X-Skip-Devui-Rewrite";

    void register(@Observes Router router) {
        String prefix = System.getenv("QUARKUS_DEV_UI_CONTEXT_ROOT");
        if (prefix == null || prefix.isBlank()) {
            return;
        }
        String normalized = prefix.endsWith("/") ? prefix.substring(0, prefix.length() - 1) : prefix;
        router.getWithRegex("/q/dev-ui/?").order(-5000).handler(ctx -> rewrite(ctx, normalized));
    }

    private static void rewrite(RoutingContext ctx, String prefix) {
        if ("1".equals(ctx.request().getHeader(SKIP))) {
            ctx.next();
            return;
        }
        String path = ctx.request().path();
        if (!path.endsWith("/")) {
            path = path + "/";
        }
        ctx.vertx().createHttpClient()
                .request(HttpMethod.GET, ctx.request().localAddress().port(), "127.0.0.1", path)
                .compose(req -> req.putHeader(SKIP, "1").send())
                .compose(resp -> resp.body().map(body -> new Object[] { resp.statusCode(),
                        resp.getHeader("Content-Type"), body.toString() }))
                .onSuccess(parts -> {
                    int status = (Integer) parts[0];
                    String type = parts[1] != null ? (String) parts[1] : "text/html; charset=utf-8";
                    String html = (String) parts[2];
                    if (status == 200 && type.contains("text/html")) {
                        html = html.replace("\"/q/_static/", "\"" + prefix + "/q/_static/");
                    }
                    ctx.response().setStatusCode(status).putHeader("Content-Type", type).end(html);
                })
                .onFailure(ctx::fail);
    }
}
