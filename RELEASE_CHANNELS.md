# 예제 채널과 릴리스

예제 코드는 `preview → rc → main` 순서로 승격합니다. 정식 버전을 사용하는 독자는 기본 브랜치인 `main`에서 시작하고, 개발 중인 예제와 릴리스 후보는 각각 `preview`와 `rc`에서 확인합니다.

## 채널별 의존성과 발행 결과

| 브랜치 | 허용하는 Spine과 CLI 버전 | 예제 태그 | GitHub Release |
| --- | --- | --- | --- |
| `preview` | 정식 버전과 개발 중인 고정 버전 | `vX.Y.Z-preview.N` | 생성하지 않음 |
| `rc` | 정식 버전 또는 `vX.Y.Z-rc.N` | `vX.Y.Z-rc.N` | 생성하지 않음 |
| `main` | 정식 버전 | `vX.Y.Z` | 생성 |

예제 태그는 이 예제 저장소의 버전입니다. 프레임워크 버전과 같다고 가정하지 마세요. 실제 Spine과 플랫폼 버전은 각 `go.mod`, CLI 버전은 [채널 설정](.github/example-channels.json)에 고정합니다. 초기 채널은 모두 기존 Spine `v0.6.5` 예제에서 시작합니다. 채널 이름을 추가해도 사전 릴리스 의존성으로 자동 변경하지 않습니다.

`rc`에서는 Preview와 기타 개발 버전을, `main`에서는 모든 사전 릴리스 버전을 거부합니다. 검사 대상 Spine과 플랫폼의 `replace`도 거부합니다. MSA의 저장소 내부 공유 모듈 연결은 허용합니다.

```sh
git clone --branch preview https://github.com/NARUBROWN/spine-user-demo.git
# RC 예제는 --branch rc, 정식 예제는 --branch main으로 선택합니다.
```

## 변경과 승격

1. 작업 브랜치에서 변경한 뒤 `preview`로 PR을 만듭니다. 외부 fork의 브랜치 이름이 `rc`나 `main`이어도 대상은 `preview`입니다.
2. Preview를 확인한 뒤 같은 저장소의 `preview → rc` PR을 merge commit으로 병합합니다.
3. RC를 확인한 뒤 같은 저장소의 `rc → main` PR을 merge commit으로 병합합니다.

단계를 건너뛰는 PR과 역방향 PR은 `Branch Flow / <대상 브랜치>` 검사에서 실패합니다. `rc`와 `main`은 최신 상태 요구 때문에 역방향 병합이 필요해지지 않도록 설정합니다. 각 PR의 merge 결과와 병합 후 브랜치 HEAD는 채널별 필수 검사로 검증합니다.

로컬에서는 대상 채널을 지정해 의존성을 검사합니다. 이 검사는 `go mod edit -json`으로 선언을 읽으며 의존성을 내려받거나 애플리케이션을 실행하지 않습니다.

```sh
python3 scripts/release/example_channels.py --channel preview
python3 scripts/release/example_channels.py --channel rc
python3 scripts/release/example_channels.py --channel main
```

사용자 API 예제에는 채널 정책 검사만 추가합니다. 기존에 제거한 애플리케이션 CI는 복구하지 않으며, 로컬 `scripts/check.sh`와 `scripts/e2e.sh`를 계속 사용합니다. 채널 검사 성공은 MySQL E2E 성공을 의미하지 않습니다.

## 예제 태그 발행

GitHub Actions의 **Publish Example**을 수동 실행합니다. Preview와 RC는 태그로만 공개하고, Stable에서 GitHub Release를 생성합니다. 브랜치 병합만으로 태그나 Release를 발행하지 않습니다.

| 입력 | 값 |
| --- | --- |
| 실행 브랜치 | `preview`, `rc`, `main` 중 발행할 단계 |
| `version` | 실행 브랜치에 맞는 새 예제 태그 |
| `source_sha` | 해당 브랜치 HEAD의 전체 40자리 SHA |
| `source_tag` | Preview는 빈칸, RC는 같은 버전의 Preview 태그, Stable은 같은 버전의 RC 태그 |

예를 들어 `v0.1.0-preview.1 → v0.1.0-rc.1 → v0.1.0` 순서로 발행합니다. 워크플로우는 요청 SHA가 현재 브랜치 HEAD인지, 해당 SHA의 필수 검사가 성공했는지 확인합니다. RC와 Stable은 이전 단계 태그가 현재 커밋의 조상이고 소스 트리가 같은지도 확인합니다. Merge commit의 SHA 차이는 허용합니다.

이전 태그와 파일이 달라졌다면 새 Preview부터 검증합니다. 의존성을 Preview에서 RC 또는 Stable 버전으로 바꾸는 경우에도 최종 의존성으로 다시 Preview를 발행한 뒤 승격해야 합니다. 기존 태그는 이동하지 않으며, 태그 생성 후 발행이 중단되면 같은 SHA로 재시도할 수 있습니다.

## 최초 브랜치 설정

처음에는 `main`만 있으므로 이 워크플로우 설정을 `main`에 먼저 반영합니다. 이는 최초 설정에만 필요한 절차입니다. 이후 변경은 `preview → rc → main` 흐름을 따릅니다.

저장소 관리자는 다음 명령으로 생성할 브랜치와 보호 규칙을 먼저 확인합니다. 기본 실행은 원격 설정을 바꾸지 않습니다.

```sh
python3 scripts/release/configure_channels.py --repository NARUBROWN/spine-user-demo
```

계획을 확인한 뒤 적용합니다.

```sh
python3 scripts/release/configure_channels.py --repository NARUBROWN/spine-user-demo --apply
```

설정 스크립트는 `main`에 워크플로우가 있는지 먼저 확인하고, 없는 `preview`와 `rc`만 `main`에서 생성합니다. 기존 채널의 커밋을 옮기지 않습니다. 기본 브랜치는 `main`으로 두고 각 채널에 PR, 채널별 필수 검사, 강제 푸시와 브랜치 삭제 금지 규칙을 설정합니다. 기존의 다른 보호 규칙은 삭제하지 않습니다.
