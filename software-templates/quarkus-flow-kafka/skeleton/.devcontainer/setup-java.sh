#!/usr/bin/env bash
# Pick JDK 21 for this app (pom maven.compiler.release=21).
# Dev Spaces UDI sets JAVA_HOME to 17; accepting that causes:
#   Fatal error compiling: error: release version 21 not supported
# Safe to `source` from the Quarkus dev command.
java_major() {
  "$1" -version 2>&1 | sed -n 's/.*version "\([0-9][0-9]*\).*/\1/p' | head -1
}

is_jdk21() {
  [ -n "${1:-}" ] && [ -x "${1}/bin/java" ] && [ "$(java_major "${1}/bin/java")" = "21" ]
}

use_jdk() {
  export JAVA_HOME="$1"
  export PATH="${JAVA_HOME}/bin:${PATH}"
  echo "JAVA_HOME=${JAVA_HOME} ($("${JAVA_HOME}/bin/java" -version 2>&1 | head -1))"
}

if is_jdk21 "${JAVA_HOME:-}"; then
  use_jdk "${JAVA_HOME}"
  return 0 2>/dev/null || exit 0
fi

JAVA_HOME=""
for d in \
  "${HOME}/.local/jdk-21" \
  /usr/lib/jvm/java-21-openjdk \
  /usr/lib/jvm/jre-21-openjdk \
  /usr/lib/jvm/java-21 \
  /usr/lib/jvm/temurin-21 \
  /usr/lib/jvm/jdk-21
do
  if is_jdk21 "${d}"; then
    use_jdk "${d}"
    return 0 2>/dev/null || exit 0
  fi
done

found="$(find /usr/lib/jvm -maxdepth 3 -type f -path '*21*/bin/java' 2>/dev/null | head -1 || true)"
if [ -n "${found}" ]; then
  candidate="$(dirname "$(dirname "${found}")")"
  if is_jdk21 "${candidate}"; then
    use_jdk "${candidate}"
    return 0 2>/dev/null || exit 0
  fi
fi

dest="${HOME}/.local/jdk-21"
if is_jdk21 "${dest}"; then
  use_jdk "${dest}"
  return 0 2>/dev/null || exit 0
fi

arch="$(uname -m)"
case "${arch}" in
  aarch64|arm64) arch_api=aarch64 ;;
  *) arch_api=x64 ;;
esac
url="https://api.adoptium.net/v3/binary/latest/21/ga/linux/${arch_api}/jdk/hotspot/normal/eclipse?project=jdk"
echo "UDI default is not JDK 21. Downloading Eclipse Temurin 21 (${arch_api}) to ${dest} ..."
tmp="$(mktemp -d)"
cleanup() { rm -rf "${tmp}"; }
trap cleanup EXIT
curl -fsSL -o "${tmp}/jdk.tgz" "${url}"
tar -xzf "${tmp}/jdk.tgz" -C "${tmp}"
extracted="$(find "${tmp}" -maxdepth 1 -mindepth 1 -type d -name 'jdk-21*' | head -1)"
if [ -z "${extracted}" ]; then
  echo "FAIL: Temurin archive did not contain jdk-21*" >&2
  ls -la "${tmp}" >&2
  return 1 2>/dev/null || exit 1
fi
mkdir -p "${HOME}/.local"
rm -rf "${dest}"
mv "${extracted}" "${dest}"
trap - EXIT
cleanup
if ! is_jdk21 "${dest}"; then
  echo "FAIL: JDK 21 is required to compile this app (maven.compiler.release=21)." >&2
  echo "      Current java: $(command -v java 2>/dev/null) $(java -version 2>&1 | head -1)" >&2
  return 1 2>/dev/null || exit 1
fi
use_jdk "${dest}"
