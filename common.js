// Code partagé par les pages de jeu (Patate, Ricochet) : icônes, connexion, son.
// Heroicons v2.2.0 (solid 24) — MIT, https://heroicons.com — [tracé SVG, evenodd] (sauf « fullscreen », dessinée ici)
const ICONS = {"fullscreen":[["M3.75 3h5.5a.75.75 0 0 1 0 1.5H5.56l4.22 4.22a.75.75 0 0 1-1.06 1.06L4.5 5.56v3.69a.75.75 0 0 1-1.5 0v-5.5A.75.75 0 0 1 3.75 3Zm11 0h5.5a.75.75 0 0 1 .75.75v5.5a.75.75 0 0 1-1.5 0V5.56l-4.22 4.22a.75.75 0 1 1-1.06-1.06l4.22-4.22h-3.69a.75.75 0 0 1 0-1.5ZM3.75 14a.75.75 0 0 1 .75.75v3.69l4.22-4.22a.75.75 0 1 1 1.06 1.06L5.56 19.5h3.69a.75.75 0 0 1 0 1.5h-5.5a.75.75 0 0 1-.75-.75v-5.5a.75.75 0 0 1 .75-.75Zm16.5 0a.75.75 0 0 1 .75.75v5.5a.75.75 0 0 1-.75.75h-5.5a.75.75 0 0 1 0-1.5h3.69l-4.22-4.22a.75.75 0 1 1 1.06-1.06l4.22 4.22v-3.69a.75.75 0 0 1 .75-.75Z",true]],"bolt":[["M14.615 1.595a.75.75 0 0 1 .359.852L12.982 9.75h7.268a.75.75 0 0 1 .548 1.262l-10.5 11.25a.75.75 0 0 1-1.272-.71l1.992-7.302H3.75a.75.75 0 0 1-.548-1.262l10.5-11.25a.75.75 0 0 1 .913-.143Z",true]],"shield-check":[["M12.516 2.17a.75.75 0 0 0-1.032 0 11.209 11.209 0 0 1-7.877 3.08.75.75 0 0 0-.722.515A12.74 12.74 0 0 0 2.25 9.75c0 5.942 4.064 10.933 9.563 12.348a.749.749 0 0 0 .374 0c5.499-1.415 9.563-6.406 9.563-12.348 0-1.39-.223-2.73-.635-3.985a.75.75 0 0 0-.722-.516l-.143.001c-2.996 0-5.717-1.17-7.734-3.08Zm3.094 8.016a.75.75 0 1 0-1.22-.872l-3.236 4.53L9.53 12.22a.75.75 0 0 0-1.06 1.06l2.25 2.25a.75.75 0 0 0 1.14-.094l3.75-5.25Z",true]],"pause-circle":[["M2.25 12c0-5.385 4.365-9.75 9.75-9.75s9.75 4.365 9.75 9.75-4.365 9.75-9.75 9.75S2.25 17.385 2.25 12ZM9 8.25a.75.75 0 0 0-.75.75v6c0 .414.336.75.75.75h.75a.75.75 0 0 0 .75-.75V9a.75.75 0 0 0-.75-.75H9Zm5.25 0a.75.75 0 0 0-.75.75v6c0 .414.336.75.75.75H15a.75.75 0 0 0 .75-.75V9a.75.75 0 0 0-.75-.75h-.75Z",true]],"fire":[["M12.963 2.286a.75.75 0 0 0-1.071-.136 9.742 9.742 0 0 0-3.539 6.176 7.547 7.547 0 0 1-1.705-1.715.75.75 0 0 0-1.152-.082A9 9 0 1 0 15.68 4.534a7.46 7.46 0 0 1-2.717-2.248ZM15.75 14.25a3.75 3.75 0 1 1-7.313-1.172c.628.465 1.35.81 2.133 1a5.99 5.99 0 0 1 1.925-3.546 3.75 3.75 0 0 1 3.255 3.718Z",true]],"user":[["M7.5 6a4.5 4.5 0 1 1 9 0 4.5 4.5 0 0 1-9 0ZM3.751 20.105a8.25 8.25 0 0 1 16.498 0 .75.75 0 0 1-.437.695A18.683 18.683 0 0 1 12 22.5c-2.786 0-5.433-.608-7.812-1.7a.75.75 0 0 1-.437-.695Z",true]],"hashtag":[["M11.097 1.515a.75.75 0 0 1 .589.882L10.666 7.5h4.47l1.079-5.397a.75.75 0 1 1 1.47.294L16.665 7.5h3.585a.75.75 0 0 1 0 1.5h-3.885l-1.2 6h3.585a.75.75 0 0 1 0 1.5h-3.885l-1.08 5.397a.75.75 0 1 1-1.47-.294l1.02-5.103h-4.47l-1.08 5.397a.75.75 0 1 1-1.47-.294l1.02-5.103H3.75a.75.75 0 0 1 0-1.5h3.885l1.2-6H5.25a.75.75 0 0 1 0-1.5h3.885l1.08-5.397a.75.75 0 0 1 .882-.588ZM10.365 9l-1.2 6h4.47l1.2-6h-4.47Z",true]],"plus-circle":[["M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25ZM12.75 9a.75.75 0 0 0-1.5 0v2.25H9a.75.75 0 0 0 0 1.5h2.25V15a.75.75 0 0 0 1.5 0v-2.25H15a.75.75 0 0 0 0-1.5h-2.25V9Z",true]],"arrow-right-end-on-rectangle":[["M16.5 3.75a1.5 1.5 0 0 1 1.5 1.5v13.5a1.5 1.5 0 0 1-1.5 1.5h-6a1.5 1.5 0 0 1-1.5-1.5V15a.75.75 0 0 0-1.5 0v3.75a3 3 0 0 0 3 3h6a3 3 0 0 0 3-3V5.25a3 3 0 0 0-3-3h-6a3 3 0 0 0-3 3V9A.75.75 0 1 0 9 9V5.25a1.5 1.5 0 0 1 1.5-1.5h6Zm-5.03 4.72a.75.75 0 0 0 0 1.06l1.72 1.72H2.25a.75.75 0 0 0 0 1.5h10.94l-1.72 1.72a.75.75 0 1 0 1.06 1.06l3-3a.75.75 0 0 0 0-1.06l-3-3a.75.75 0 0 0-1.06 0Z",true]],"exclamation-triangle":[["M9.401 3.003c1.155-2 4.043-2 5.197 0l7.355 12.748c1.154 2-.29 4.5-2.599 4.5H4.645c-2.309 0-3.752-2.5-2.598-4.5L9.4 3.003ZM12 8.25a.75.75 0 0 1 .75.75v3.75a.75.75 0 0 1-1.5 0V9a.75.75 0 0 1 .75-.75Zm0 8.25a.75.75 0 1 0 0-1.5.75.75 0 0 0 0 1.5Z",true]],"clock":[["M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25ZM12.75 6a.75.75 0 0 0-1.5 0v6c0 .414.336.75.75.75h4.5a.75.75 0 0 0 0-1.5h-3.75V6Z",true]],"star":[["M10.788 3.21c.448-1.077 1.976-1.077 2.424 0l2.082 5.006 5.404.434c1.164.093 1.636 1.545.749 2.305l-4.117 3.527 1.257 5.273c.271 1.136-.964 2.033-1.96 1.425L12 18.354 7.373 21.18c-.996.608-2.231-.29-1.96-1.425l1.257-5.273-4.117-3.527c-.887-.76-.415-2.212.749-2.305l5.404-.434 2.082-5.005Z",true]],"trophy":[["M5.166 2.621v.858c-1.035.148-2.059.33-3.071.543a.75.75 0 0 0-.584.859 6.753 6.753 0 0 0 6.138 5.6 6.73 6.73 0 0 0 2.743 1.346A6.707 6.707 0 0 1 9.279 15H8.54c-1.036 0-1.875.84-1.875 1.875V19.5h-.75a2.25 2.25 0 0 0-2.25 2.25c0 .414.336.75.75.75h15a.75.75 0 0 0 .75-.75 2.25 2.25 0 0 0-2.25-2.25h-.75v-2.625c0-1.036-.84-1.875-1.875-1.875h-.739a6.706 6.706 0 0 1-1.112-3.173 6.73 6.73 0 0 0 2.743-1.347 6.753 6.753 0 0 0 6.139-5.6.75.75 0 0 0-.585-.858 47.077 47.077 0 0 0-3.07-.543V2.62a.75.75 0 0 0-.658-.744 49.22 49.22 0 0 0-6.093-.377c-2.063 0-4.096.128-6.093.377a.75.75 0 0 0-.657.744Zm0 2.629c0 1.196.312 2.32.857 3.294A5.266 5.266 0 0 1 3.16 5.337a45.6 45.6 0 0 1 2.006-.343v.256Zm13.5 0v-.256c.674.1 1.343.214 2.006.343a5.265 5.265 0 0 1-2.863 3.207 6.72 6.72 0 0 0 .857-3.294Z",true]],"cpu-chip":[["M16.5 7.5h-9v9h9v-9Z",false],["M8.25 2.25A.75.75 0 0 1 9 3v.75h2.25V3a.75.75 0 0 1 1.5 0v.75H15V3a.75.75 0 0 1 1.5 0v.75h.75a3 3 0 0 1 3 3v.75H21A.75.75 0 0 1 21 9h-.75v2.25H21a.75.75 0 0 1 0 1.5h-.75V15H21a.75.75 0 0 1 0 1.5h-.75v.75a3 3 0 0 1-3 3h-.75V21a.75.75 0 0 1-1.5 0v-.75h-2.25V21a.75.75 0 0 1-1.5 0v-.75H9V21a.75.75 0 0 1-1.5 0v-.75h-.75a3 3 0 0 1-3-3v-.75H3A.75.75 0 0 1 3 15h.75v-2.25H3a.75.75 0 0 1 0-1.5h.75V9H3a.75.75 0 0 1 0-1.5h.75v-.75a3 3 0 0 1 3-3h.75V3a.75.75 0 0 1 .75-.75ZM6 6.75A.75.75 0 0 1 6.75 6h10.5a.75.75 0 0 1 .75.75v10.5a.75.75 0 0 1-.75.75H6.75a.75.75 0 0 1-.75-.75V6.75Z",true]],"user-plus":[["M5.25 6.375a4.125 4.125 0 1 1 8.25 0 4.125 4.125 0 0 1-8.25 0ZM2.25 19.125a7.125 7.125 0 0 1 14.25 0v.003l-.001.119a.75.75 0 0 1-.363.63 13.067 13.067 0 0 1-6.761 1.873c-2.472 0-4.786-.684-6.76-1.873a.75.75 0 0 1-.364-.63l-.001-.122ZM18.75 7.5a.75.75 0 0 0-1.5 0v2.25H15a.75.75 0 0 0 0 1.5h2.25v2.25a.75.75 0 0 0 1.5 0v-2.25H21a.75.75 0 0 0 0-1.5h-2.25V7.5Z",false]],"user-minus":[["M10.375 2.25a4.125 4.125 0 1 0 0 8.25 4.125 4.125 0 0 0 0-8.25ZM10.375 12a7.125 7.125 0 0 0-7.124 7.247.75.75 0 0 0 .363.63 13.067 13.067 0 0 0 6.761 1.873c2.472 0 4.786-.684 6.76-1.873a.75.75 0 0 0 .364-.63l.001-.12v-.002A7.125 7.125 0 0 0 10.375 12ZM16 9.75a.75.75 0 0 0 0 1.5h6a.75.75 0 0 0 0-1.5h-6Z",false]],"link":[["M19.902 4.098a3.75 3.75 0 0 0-5.304 0l-4.5 4.5a3.75 3.75 0 0 0 1.035 6.037.75.75 0 0 1-.646 1.353 5.25 5.25 0 0 1-1.449-8.45l4.5-4.5a5.25 5.25 0 1 1 7.424 7.424l-1.757 1.757a.75.75 0 1 1-1.06-1.06l1.757-1.757a3.75 3.75 0 0 0 0-5.304Zm-7.389 4.267a.75.75 0 0 1 1-.353 5.25 5.25 0 0 1 1.449 8.45l-4.5 4.5a5.25 5.25 0 1 1-7.424-7.424l1.757-1.757a.75.75 0 1 1 1.06 1.06l-1.757 1.757a3.75 3.75 0 1 0 5.304 5.304l4.5-4.5a3.75 3.75 0 0 0-1.035-6.037.75.75 0 0 1-.354-1Z",true]],"check":[["M19.916 4.626a.75.75 0 0 1 .208 1.04l-9 13.5a.75.75 0 0 1-1.154.114l-6-6a.75.75 0 0 1 1.06-1.06l5.353 5.353 8.493-12.74a.75.75 0 0 1 1.04-.207Z",true]],"chart-bar-square":[["M3 6a3 3 0 0 1 3-3h12a3 3 0 0 1 3 3v12a3 3 0 0 1-3 3H6a3 3 0 0 1-3-3V6Zm4.5 7.5a.75.75 0 0 1 .75.75v2.25a.75.75 0 0 1-1.5 0v-2.25a.75.75 0 0 1 .75-.75Zm3.75-1.5a.75.75 0 0 0-1.5 0v4.5a.75.75 0 0 0 1.5 0V12Zm2.25-3a.75.75 0 0 1 .75.75v6.75a.75.75 0 0 1-1.5 0V9.75A.75.75 0 0 1 13.5 9Zm3.75-1.5a.75.75 0 0 0-1.5 0v9a.75.75 0 0 0 1.5 0v-9Z",true]],"map":[["M8.161 2.58a1.875 1.875 0 0 1 1.678 0l4.993 2.498c.106.052.23.052.336 0l3.869-1.935A1.875 1.875 0 0 1 21.75 4.82v12.485c0 .71-.401 1.36-1.037 1.677l-4.875 2.437a1.875 1.875 0 0 1-1.676 0l-4.994-2.497a.375.375 0 0 0-.336 0l-3.868 1.935A1.875 1.875 0 0 1 2.25 19.18V6.695c0-.71.401-1.36 1.036-1.677l4.875-2.437ZM9 6a.75.75 0 0 1 .75.75V15a.75.75 0 0 1-1.5 0V6.75A.75.75 0 0 1 9 6Zm6.75 3a.75.75 0 0 0-1.5 0v8.25a.75.75 0 0 0 1.5 0V9Z",true]],"face-frown":[["M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm-2.625 6c-.54 0-.828.419-.936.634a1.96 1.96 0 0 0-.189.866c0 .298.059.605.189.866.108.215.395.634.936.634.54 0 .828-.419.936-.634.13-.26.189-.568.189-.866 0-.298-.059-.605-.189-.866-.108-.215-.395-.634-.936-.634Zm4.314.634c.108-.215.395-.634.936-.634.54 0 .828.419.936.634.13.26.189.568.189.866 0 .298-.059.605-.189.866-.108.215-.395.634-.936.634-.54 0-.828-.419-.936-.634a1.96 1.96 0 0 1-.189-.866c0-.298.059-.605.189-.866Zm-4.34 7.964a.75.75 0 0 1-1.061-1.06 5.236 5.236 0 0 1 3.73-1.538 5.236 5.236 0 0 1 3.695 1.538.75.75 0 1 1-1.061 1.06 3.736 3.736 0 0 0-2.639-1.098 3.736 3.736 0 0 0-2.664 1.098Z",true]],"arrow-path":[["M4.755 10.059a7.5 7.5 0 0 1 12.548-3.364l1.903 1.903h-3.183a.75.75 0 1 0 0 1.5h4.992a.75.75 0 0 0 .75-.75V4.356a.75.75 0 0 0-1.5 0v3.18l-1.9-1.9A9 9 0 0 0 3.306 9.67a.75.75 0 1 0 1.45.388Zm15.408 3.352a.75.75 0 0 0-.919.53 7.5 7.5 0 0 1-12.548 3.364l-1.902-1.903h3.183a.75.75 0 0 0 0-1.5H2.984a.75.75 0 0 0-.75.75v4.992a.75.75 0 0 0 1.5 0v-3.18l1.9 1.9a9 9 0 0 0 15.059-4.035.75.75 0 0 0-.53-.918Z",true]],"sparkles":[["M9 4.5a.75.75 0 0 1 .721.544l.813 2.846a3.75 3.75 0 0 0 2.576 2.576l2.846.813a.75.75 0 0 1 0 1.442l-2.846.813a3.75 3.75 0 0 0-2.576 2.576l-.813 2.846a.75.75 0 0 1-1.442 0l-.813-2.846a3.75 3.75 0 0 0-2.576-2.576l-2.846-.813a.75.75 0 0 1 0-1.442l2.846-.813A3.75 3.75 0 0 0 7.466 7.89l.813-2.846A.75.75 0 0 1 9 4.5ZM18 1.5a.75.75 0 0 1 .728.568l.258 1.036c.236.94.97 1.674 1.91 1.91l1.036.258a.75.75 0 0 1 0 1.456l-1.036.258c-.94.236-1.674.97-1.91 1.91l-.258 1.036a.75.75 0 0 1-1.456 0l-.258-1.036a2.625 2.625 0 0 0-1.91-1.91l-1.036-.258a.75.75 0 0 1 0-1.456l1.036-.258a2.625 2.625 0 0 0 1.91-1.91l.258-1.036A.75.75 0 0 1 18 1.5ZM16.5 15a.75.75 0 0 1 .712.513l.394 1.183c.15.447.5.799.948.948l1.183.395a.75.75 0 0 1 0 1.422l-1.183.395c-.447.15-.799.5-.948.948l-.395 1.183a.75.75 0 0 1-1.422 0l-.395-1.183a1.5 1.5 0 0 0-.948-.948l-1.183-.395a.75.75 0 0 1 0-1.422l1.183-.395c.447-.15.799-.5.948-.948l.395-1.183A.75.75 0 0 1 16.5 15Z",true]],"chevron-right":[["M16.28 11.47a.75.75 0 0 1 0 1.06l-7.5 7.5a.75.75 0 0 1-1.06-1.06L14.69 12 7.72 5.03a.75.75 0 0 1 1.06-1.06l7.5 7.5Z",true]],"chevron-left":[["M7.72 12.53a.75.75 0 0 1 0-1.06l7.5-7.5a.75.75 0 1 1 1.06 1.06L9.31 12l6.97 6.97a.75.75 0 1 1-1.06 1.06l-7.5-7.5Z",true]],"petanque":[["M4 14.5a5 5 0 1 0 10 0a5 5 0 1 0-10 0Z",false],["M14.5 16a3.75 3.75 0 1 0 7.5 0a3.75 3.75 0 1 0-7.5 0Z",false],["M15 6.5a2 2 0 1 0 4 0a2 2 0 1 0-4 0Z",false]],"billard":[["M12 2.25a9.75 9.75 0 1 0 0 19.5a9.75 9.75 0 1 0 0-19.5ZM12 8a4 4 0 1 1 0 8a4 4 0 1 1 0-8Z",true],["M10.5 12a1.5 1.5 0 1 0 3 0a1.5 1.5 0 1 0-3 0Z",false]],"tank":[["M2.5 15.5h19v2a2.5 2.5 0 0 1-2.5 2.5H5a2.5 2.5 0 0 1-2.5-2.5v-2Z",false],["M6.5 10.5h8l2 4h-12l2-4Z",false],["M13.6 11.3l7.3-4.2.8 1.3-7.3 4.2-.8-1.3Z",false]],"trail":[["M3 19.5v-3h6v-7.5h6.5v-4.5h5.5v3h-2.5v4.5h-6.5v7.5H3Z",false]],"worm":[["M2.5 17a2.5 2.5 0 1 0 5 0a2.5 2.5 0 1 0-5 0Z",false],["M6 15a2.75 2.75 0 1 0 5.5 0a2.75 2.75 0 1 0-5.5 0Z",false],["M10 13a3 3 0 1 0 6 0a3 3 0 1 0-6 0Z",false],["M14 9a3.75 3.75 0 1 0 7.5 0a3.75 3.75 0 1 0-7.5 0Z",false]],"heart":[["m11.645 20.91-.007-.003-.022-.012a15.247 15.247 0 0 1-.383-.218 25.18 25.18 0 0 1-4.244-3.17C4.688 15.36 2.25 12.174 2.25 8.25 2.25 5.322 4.714 3 7.688 3A5.5 5.5 0 0 1 12 5.052 5.5 5.5 0 0 1 16.313 3c2.973 0 5.437 2.322 5.437 5.25 0 3.925-2.438 7.111-4.739 9.256a25.175 25.175 0 0 1-4.244 3.17 15.247 15.247 0 0 1-.383.219l-.022.012-.007.004-.003.001a.752.752 0 0 1-.704 0l-.003-.001Z",false]],"speaker-wave":[["M13.5 4.06c0-1.336-1.616-2.005-2.56-1.06l-4.5 4.5H4.508c-1.141 0-2.318.664-2.66 1.905A9.76 9.76 0 0 0 1.5 12c0 .898.121 1.768.35 2.595.341 1.24 1.518 1.905 2.659 1.905h1.93l4.5 4.5c.945.945 2.561.276 2.561-1.06V4.06ZM18.584 5.106a.75.75 0 0 1 1.06 0c3.808 3.807 3.808 9.98 0 13.788a.75.75 0 0 1-1.06-1.06 8.25 8.25 0 0 0 0-11.668.75.75 0 0 1 0-1.06Z",false],["M15.932 7.757a.75.75 0 0 1 1.061 0 6 6 0 0 1 0 8.486.75.75 0 0 1-1.06-1.061 4.5 4.5 0 0 0 0-6.364.75.75 0 0 1 0-1.06Z",false]],"speaker-x-mark":[["M13.5 4.06c0-1.336-1.616-2.005-2.56-1.06l-4.5 4.5H4.508c-1.141 0-2.318.664-2.66 1.905A9.76 9.76 0 0 0 1.5 12c0 .898.121 1.768.35 2.595.341 1.24 1.518 1.905 2.659 1.905h1.93l4.5 4.5c.945.945 2.561.276 2.561-1.06V4.06ZM17.78 9.22a.75.75 0 1 0-1.06 1.06L18.44 12l-1.72 1.72a.75.75 0 1 0 1.06 1.06l1.72-1.72 1.72 1.72a.75.75 0 1 0 1.06-1.06L20.56 12l1.72-1.72a.75.75 0 1 0-1.06-1.06l-1.72 1.72-1.72-1.72Z",false]],"users":[["M4.5 6.375a4.125 4.125 0 1 1 8.25 0 4.125 4.125 0 0 1-8.25 0ZM14.25 8.625a3.375 3.375 0 1 1 6.75 0 3.375 3.375 0 0 1-6.75 0ZM1.5 19.125a7.125 7.125 0 0 1 14.25 0v.003l-.001.119a.75.75 0 0 1-.363.63 13.067 13.067 0 0 1-6.761 1.873c-2.472 0-4.786-.684-6.76-1.873a.75.75 0 0 1-.364-.63l-.001-.122ZM17.25 19.128l-.001.144a2.25 2.25 0 0 1-.233.96 10.088 10.088 0 0 0 5.06-1.01.75.75 0 0 0 .42-.643 4.875 4.875 0 0 0-6.957-4.611 8.586 8.586 0 0 1 1.71 5.157v.003Z",false]],"tv":[["M2.25 5.5A2.5 2.5 0 0 1 4.75 3h14.5a2.5 2.5 0 0 1 2.5 2.5v9.5a2.5 2.5 0 0 1-2.5 2.5H4.75a2.5 2.5 0 0 1-2.5-2.5V5.5Z",false],["M7.5 19.5h9a.75.75 0 0 1 0 1.5h-9a.75.75 0 0 1 0-1.5Z",false]],"phone":[["M7 3.5A2.5 2.5 0 0 1 9.5 1h5A2.5 2.5 0 0 1 17 3.5v17a2.5 2.5 0 0 1-2.5 2.5h-5A2.5 2.5 0 0 1 7 20.5v-17ZM10.5 19.25a.75.75 0 0 0 0 1.5h3a.75.75 0 0 0 0-1.5h-3Z",true]],"sumo":[["M2 13a5.25 5.25 0 1 0 10.5 0a5.25 5.25 0 1 0-10.5 0Z",false],["M12.75 10.5a4.6 4.6 0 1 0 9.2 0a4.6 4.6 0 1 0-9.2 0Z",false]],"quiz":[["M4.5 3h4.5A1.5 1.5 0 0 1 10.5 4.5V9A1.5 1.5 0 0 1 9 10.5H4.5A1.5 1.5 0 0 1 3 9V4.5A1.5 1.5 0 0 1 4.5 3ZM15 3h4.5A1.5 1.5 0 0 1 21 4.5V9a1.5 1.5 0 0 1-1.5 1.5H15A1.5 1.5 0 0 1 13.5 9V4.5A1.5 1.5 0 0 1 15 3ZM4.5 13.5H9a1.5 1.5 0 0 1 1.5 1.5v4.5A1.5 1.5 0 0 1 9 21H4.5A1.5 1.5 0 0 1 3 19.5V15a1.5 1.5 0 0 1 1.5-1.5ZM15 13.5h4.5A1.5 1.5 0 0 1 21 15v4.5a1.5 1.5 0 0 1-1.5 1.5H15a1.5 1.5 0 0 1-1.5-1.5V15a1.5 1.5 0 0 1 1.5-1.5Z",false]],"mask":[["M12 2.25a9.75 9.75 0 1 0 0 19.5a9.75 9.75 0 1 0 0-19.5ZM6.75 10.5c.6-1.2 1.6-1.8 2.75-1.8s2.15.6 2.25 1.8c-1.4.9-3.6.9-5 0Zm10.5 0c-1.4.9-3.6.9-5 0c.1-1.2 1.1-1.8 2.25-1.8s2.15.6 2.75 1.8ZM8.25 15.75c2.2 1.4 5.3 1.4 7.5 0c-.6 1.9-2 3-3.75 3s-3.15-1.1-3.75-3Z",true]],"car":[["M4 12.5l1.7-4.3A2.25 2.25 0 0 1 7.8 6.75h8.4a2.25 2.25 0 0 1 2.1 1.45L20 12.5v5a1 1 0 0 1-1 1h-1.75a1 1 0 0 1-1-1v-1h-8.5v1a1 1 0 0 1-1 1H5a1 1 0 0 1-1-1v-5ZM7.4 8.9 6.5 11.5h11l-.9-2.6a.75.75 0 0 0-.7-.5H8.1a.75.75 0 0 0-.7.5ZM7 13.25a1.1 1.1 0 1 0 0 2.2a1.1 1.1 0 1 0 0-2.2ZM17 13.25a1.1 1.1 0 1 0 0 2.2a1.1 1.1 0 1 0 0-2.2Z",true]],"crown":[["M2.75 7.5l4.75 3.75L12 4.5l4.5 6.75 4.75-3.75-2 11.25H4.75L2.75 7.5Z",false]],"face-smile":[["M12 2.25c-5.385 0-9.75 4.365-9.75 9.75s4.365 9.75 9.75 9.75 9.75-4.365 9.75-9.75S17.385 2.25 12 2.25Zm-2.625 6c-.54 0-.828.419-.936.634a1.96 1.96 0 0 0-.189.866c0 .298.059.605.189.866.108.215.395.634.936.634.54 0 .828-.419.936-.634.13-.26.189-.568.189-.866 0-.298-.059-.605-.189-.866-.108-.215-.395-.634-.936-.634Zm4.314.634c.108-.215.395-.634.936-.634.54 0 .828.419.936.634.13.26.189.568.189.866 0 .298-.059.605-.189.866-.108.215-.395.634-.936.634-.54 0-.828-.419-.936-.634a1.96 1.96 0 0 1-.189-.866c0-.298.059-.605.189-.866Zm2.023 6.828a.75.75 0 1 0-1.06-1.06 3.75 3.75 0 0 1-5.304 0 .75.75 0 0 0-1.06 1.06 5.25 5.25 0 0 0 7.424 0Z",true]]};

// Icônes au trait (LINES) : Lucide v0.460.0 — https://lucide.dev — sauf shotgun, smg, bat, dynamite, mine et heli,
// dessinées ici dans le même style. Licence de Lucide :
//   ISC License. Copyright (c) for portions of Lucide are held by Cole Bemis 2013-2022 as part of Feather (MIT).
//   All other copyright (c) for Lucide are held by Lucide Contributors 2022.
//   Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby
//   granted, provided that the above copyright notice and this permission notice appear in all copies.
//   THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING
//   ALL IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL,
//   DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS,
//   WHETHER IN AN ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE
//   USE OR PERFORMANCE OF THIS SOFTWARE.
const LINES = {"rocket":["M4.5 16.5c-1.5 1.26-2 5-2 5s3.74-.5 5-2c.71-.84.7-2.13-.09-2.91a2.18 2.18 0 0 0-2.91-.09z","m12 15-3-3a22 22 0 0 1 2-3.95A12.88 12.88 0 0 1 22 2c0 2.72-.78 7.5-6 11a22.35 22.35 0 0 1-4 2z","M9 12H4s.55-3.03 2-4c1.62-1.08 5 0 5 0","M12 15v5s3.03-.55 4-2c1.08-1.62 0-5 0-5"],"bomb":["M2 13a9 9 0 1 0 18 0a9 9 0 1 0 -18 0","M14.35 4.65 16.3 2.7a2.41 2.41 0 0 1 3.4 0l1.6 1.6a2.4 2.4 0 0 1 0 3.4l-1.95 1.95","m22 2-1.5 1.5"],"flame":["M8.5 14.5A2.5 2.5 0 0 0 11 12c0-1.38-.5-2-1-3-1.072-2.143-.224-4.054 2-6 .5 2.5 2 4.9 4 6.5 2 1.6 3 3.5 3 5.5a7 7 0 1 1-14 0c0-1.153.433-2.294 1-3a2.5 2.5 0 0 0 2.5 2.5z"],"plane":["M17.8 19.2 16 11l3.5-3.5C21 6 21.5 4 21 3c-1-.5-3 0-4.5 1.5L13 8 4.8 6.2c-.5-.1-.9.1-1.1.5l-.3.5c-.2.5-.1 1 .3 1.3L9 12l-2 3H4l-1 1 3 2 2 3 1-1v-3l3-2 3.5 5.3c.3.4.8.5 1.3.3l.5-.2c.4-.3.6-.7.5-1.2z"],"lasso":["M7 22a5 5 0 0 1-2-4","M3.3 14A6.8 6.8 0 0 1 2 10c0-4.4 4.5-8 10-8s10 3.6 10 8-4.5 8-10 8a12 12 0 0 1-5-1","M5 18a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"],"crosshair":["M2 12a10 10 0 1 0 20 0a10 10 0 1 0 -20 0","M22 12L18 12","M6 12L2 12","M12 6L12 2","M12 22L12 18"],"swords":["M14.5 17.5L3 6L3 3L6 3L17.5 14.5","M13 19L19 13","M16 16L20 20","M19 21L21 19","M14.5 6.5L18 3L21 3L21 6L17.5 9.5","M5 14L9 18","M7 17L4 20","M3 19L5 21"],"zap":["M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"],"move":["M12 2v20","m15 19-3 3-3-3","m19 9 3 3-3 3","M2 12h20","m5 9-3 3 3 3","m9 5 3-3 3 3"],"mountain":["m8 3 4 8 5-5 5 15H2L8 3z"],"castle":["M22 20v-9H2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2Z","M18 11V4H6v7","M15 22v-4a3 3 0 0 0-3-3a3 3 0 0 0-3 3v4","M22 11V9","M2 11V9","M6 4V2","M18 4V2","M10 4V2","M14 4V2"],"palm":["M13 8c0-2.76-2.46-5-5.5-5S2 5.24 2 8h2l1-1 1 1h4","M13 7.14A5.82 5.82 0 0 1 16.5 6c3.04 0 5.5 2.24 5.5 5h-3l-1-1-1 1h-3","M5.89 9.71c-2.15 2.15-2.3 5.47-.35 7.43l4.24-4.25.7-.7.71-.71 2.12-2.12c-1.95-1.96-5.27-1.8-7.42.35","M11 15.5c.5 2.5-.17 4.5-1 6.5h4c2-5.5-.5-12-1-14"],"snow-mountain":["m8 3 4 8 5-5 5 15H2L8 3z","M4.14 15.08c2.62-1.57 5.24-1.43 7.86.42 2.74 1.94 5.49 2 8.23.19"],"gem":["M6 3h12l4 6-10 13L2 9Z","M11 3 8 9l4 13 4-13-3-6","M2 9h20"],"city":["M6 22V4a2 2 0 0 1 2-2h8a2 2 0 0 1 2 2v18Z","M6 12H4a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h2","M18 9h2a2 2 0 0 1 2 2v9a2 2 0 0 1-2 2h-2","M10 6h4","M10 10h4","M10 14h4","M10 18h4"],"waves":["M2 6c.6.5 1.2 1 2.5 1C7 7 7 5 9.5 5c2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1","M2 12c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1","M2 18c.6.5 1.2 1 2.5 1 2.5 0 2.5-2 5-2 2.6 0 2.4 2 5 2 2.5 0 2.5-2 5-2 1.3 0 1.9.5 2.5 1"],"chat":["M7.9 20A9 9 0 1 0 4 16.1L2 22Z"],"camera":["M14.5 4h-5L7 7H4a2 2 0 0 0-2 2v9a2 2 0 0 0 2 2h16a2 2 0 0 0 2-2V9a2 2 0 0 0-2-2h-3l-2.5-3z","M9 13a3 3 0 1 0 6 0a3 3 0 1 0 -6 0"],"pencil":["M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z","m15 5 4 4"],"check-circle":["M2 12a10 10 0 1 0 20 0a10 10 0 1 0 -20 0","m9 12 2 2 4-4"],"x":["M18 6 6 18","m6 6 12 12"],"wind":["M12.8 19.6A2 2 0 1 0 14 16H2","M17.5 8a2.5 2.5 0 1 1 2 4H2","M9.8 4.4A2 2 0 1 1 11 8H2"],"laugh":["M2 12a10 10 0 1 0 20 0a10 10 0 1 0 -20 0","M18 13a6 6 0 0 1-6 5 6 6 0 0 1-6-5h12Z","M9 9L9.01 9","M15 9L15.01 9"],"angry":["M2 12a10 10 0 1 0 20 0a10 10 0 1 0 -20 0","M16 16s-1.5-2-4-2-4 2-4 2","M7.5 8 10 9","m14 9 2.5-1","M9 10h.01","M15 10h.01"],"hand":["M18 11V6a2 2 0 0 0-2-2a2 2 0 0 0-2 2","M14 10V4a2 2 0 0 0-2-2a2 2 0 0 0-2 2v2","M10 10.5V6a2 2 0 0 0-2-2a2 2 0 0 0-2 2v8","M18 8a2 2 0 1 1 4 0v6a8 8 0 0 1-8 8h-2c-2.8 0-4.5-.86-5.99-2.34l-3.6-3.6a2 2 0 0 1 2.83-2.82L7 15"],"skull":["m12.5 17-.5-1-.5 1h1z","M15 22a1 1 0 0 0 1-1v-1a2 2 0 0 0 1.56-3.25 8 8 0 1 0-11.12 0A2 2 0 0 0 8 20v1a1 1 0 0 0 1 1z","M14 12a1 1 0 1 0 2 0a1 1 0 1 0 -2 0","M8 12a1 1 0 1 0 2 0a1 1 0 1 0 -2 0"],"shotgun":["M2 9.5h14l1.5-1.5H22v3.5h-6l-1.5 1.5H9.5l-1.5 4.5H5l1-4.5H2z","M10 13v2.5h4V13"],"smg":["M3 8h15v4h-7l-1 2H7l1-2H3z","M18 9.5h3","M12 12l1 7h3l-1-7"],"bat":["M5 17.5 16 5.5a3 3 0 0 1 4.5 4L8.5 20z","M5 17.5 3.5 19","M8.5 20 7 21.5","M2.5 20l1.5 1.5"],"dynamite":["M4 11h4v10H4z","M10 11h4v10h-4z","M16 11h4v10h-4z","M4 15h16","M12 11c0-3 1.5-5 4-6","M19 2v2","M21 4h-2","M20.5 6.5 19 5"],"mine":["M3 19h18","M5 19a7 7 0 0 1 14 0","M12 12V8","M10 6a2 2 0 1 0 4 0a2 2 0 1 0-4 0","M8 15.5h.01","M12 14.5h.01","M16 15.5h.01"],"heli":["M3 4h16","M11 4v3","M5 12a5 5 0 0 1 5-5h2a4 4 0 0 1 4 4v1a3 3 0 0 1-3 3H8a3 3 0 0 1-3-3z","M16 11h5","M21 9v4","M6 20h10","M8 15v5","M13 15v5"]};

const NS = 'http://www.w3.org/2000/svg';
// icon : une icône en SVG, pleine (ICONS) ou au trait (LINES), de la couleur du texte.
function icon(name, cls = 'size-5') {
  const s = document.createElementNS(NS, 'svg'), line = LINES[name];
  s.setAttribute('viewBox', '0 0 24 24'); s.setAttribute('aria-hidden', 'true');
  s.setAttribute('class', 'shrink-0 ' + cls);
  if (line) for (const [k, v] of [['fill', 'none'], ['stroke', 'currentColor'], ['stroke-width', '2'], ['stroke-linecap', 'round'], ['stroke-linejoin', 'round']]) s.setAttribute(k, v);
  else s.setAttribute('fill', 'currentColor');
  for (const [d, eo] of line ? line.map(d => [d]) : ICONS[name]) {
    const p = document.createElementNS(NS, 'path');
    p.setAttribute('d', d);
    if (eo) { p.setAttribute('fill-rule', 'evenodd'); p.setAttribute('clip-rule', 'evenodd'); }
    s.append(p);
  }
  return s;
}
document.querySelectorAll('i[data-icon]').forEach(i => {
  const s = icon(i.dataset.icon, i.className);
  if (i.id) s.id = i.id;
  if (i.hidden) s.setAttribute('hidden', '');
  i.replaceWith(s);
});

const $ = id => document.getElementById(id);

// lineIcon dessine une icône au trait (LINES) dans un canvas, centrée en (x, y), de size pixels.
const linePaths = {};
function lineIcon(ctx, name, x, y, size, color) {
  ctx.save(); ctx.translate(x - size / 2, y - size / 2); ctx.scale(size / 24, size / 24);
  ctx.strokeStyle = color; ctx.lineWidth = 2; ctx.lineCap = 'round'; ctx.lineJoin = 'round';
  for (const p of linePaths[name] ||= LINES[name].map(d => new Path2D(d))) ctx.stroke(p);
  ctx.restore();
}

// ---------- Menu des jeux, barre des jeux (accueil) et bouton Quitter (en partie) : une seule liste pour tout le site ----------
// classes écrites en entier : Tailwind ne génère que les noms complets qu'il trouve dans ce fichier
// min, max : joueurs humains (min 1 quand des bots complètent), pour les conseils du salon
const GAMES = [
  { href: '/patate.html', min: 1, max: 10, name: 'Patate', icon: 'fire', players: '2 à 10', tag: "Passe la bombe avant qu'elle n'explose.",
    text: 'text-orange-400', tile: 'bg-orange-500' },
  { href: '/ricochet.html', min: 1, max: 2, name: 'Ricochet', icon: 'bolt', players: '2', tag: 'Duel de raquettes, ou contre un bot.',
    text: 'text-cyan-400', tile: 'bg-cyan-500' },
  { href: '/golf.html', min: 1, max: 8, name: 'Golf', icon: 'map', players: '1 à 8', tag: 'Mini-golf, 9 trous difficiles, tous en même temps.',
    text: 'text-emerald-400', tile: 'bg-emerald-500' },
  { href: '/dessin.html', min: 3, max: 10, name: 'Dessine', icon: 'sparkles', players: '3 à 10', tag: "Quelqu'un dessine ta photo, les autres devinent.",
    text: 'text-pink-400', tile: 'bg-pink-500' },
  { href: '/foot.html', min: 1, max: 2, name: 'Foot', icon: 'star', players: '2', tag: 'Foot de pions chacun son tour, premier à 3 buts.',
    text: 'text-lime-400', tile: 'bg-lime-500' },
  { href: '/petanque.html', min: 1, max: 6, name: 'Pétanque', icon: 'petanque', players: '2 à 6', tag: 'Chacun ses 3 boules, au plus près du cochonnet. Bots possibles.',
    text: 'text-yellow-400', tile: 'bg-yellow-500' },
  { href: '/billard.html', min: 1, max: 2, name: 'Billard', icon: 'billard', players: '2', tag: 'Jaunes contre rouges, la noire pour finir. Ou contre un bot.',
    text: 'text-sky-400', tile: 'bg-sky-500' },
  { href: '/artillerie.html', min: 1, max: 4, name: 'Artillerie', icon: 'tank', players: '2 à 4', tag: 'Chars, vent et cratères : le dernier debout gagne.',
    text: 'text-red-400', tile: 'bg-red-500' },
  { href: '/motos.html', min: 1, max: 8, name: 'Motos', icon: 'trail', players: '2 à 8', tag: 'Ne touche aucun mur, pas même le tien. Bots possibles.',
    text: 'text-violet-400', tile: 'bg-violet-500' },
  { href: '/lombrics.html', min: 1, max: 4, name: 'Lombrics', icon: 'worm', players: '2 à 4', tag: 'Équipes de vers, 11 armes et 6 cartes à creuser : château, plage, ville…',
    text: 'text-pink-400', tile: 'bg-pink-500' },
];
// Mode télé : la télé affiche le jeu, les téléphones sont les manettes. Les jeux se lancent depuis la manette du chef.
const TV_GAMES = [
  { id: 'sumo', name: 'Sumo', icon: 'sumo', players: '1 à 10', min: 1, tag: "Pousse les autres hors d'une arène qui rétrécit. Bots possibles.",
    text: 'text-rose-400', tile: 'bg-rose-500' },
  { id: 'course', name: 'Course', icon: 'car', players: '1 à 10', min: 1, tag: "Petites voitures, turbo et flaques d'huile, 3 tours. Bots possibles.",
    text: 'text-sky-400', tile: 'bg-sky-500' },
  { id: 'quiz', name: 'Quiz rapide', icon: 'quiz', players: '1 à 10', min: 1, tag: 'Réponds plus vite que les autres, 10 questions.',
    text: 'text-indigo-400', tile: 'bg-indigo-500' },
  { id: 'imposteur', name: 'Imposteur', icon: 'mask', players: '3 à 10', min: 3, tag: 'Tout le monde a le même mot secret, sauf un. Démasquez-le.',
    text: 'text-teal-400', tile: 'bg-teal-500' },
  { id: 'chaine', name: 'Dessin en chaîne', icon: 'link', players: '3 à 10', min: 3, tag: 'Une phrase, un dessin, une devinette… et on rit à la fin.',
    text: 'text-amber-400', tile: 'bg-amber-500' },
];
function menuRow(href, g) {
  const a = document.createElement('a'), tile = document.createElement('span'), body = document.createElement('span'),
    txt = document.createElement('span'), name = document.createElement('span'), tag = document.createElement('span');
  a.href = href;
  a.className = 'group flex items-center gap-3 pl-4 transition-colors hover:bg-white/5 active:bg-white/10';
  tile.className = 'grid size-[30px] shrink-0 place-items-center rounded-[7px] text-white ' + g.tile;
  tile.append(icon(g.icon, 'size-[19px]'));
  body.className = 'flex min-w-0 flex-1 items-center gap-3 border-b border-white/10 py-3 pr-4 group-last:border-0';
  name.className = 'block text-[17px] leading-snug'; name.textContent = g.name;
  tag.className = 'mt-0.5 block text-[13px] text-slate-400'; tag.textContent = g.tag;
  txt.className = 'min-w-0 flex-1'; txt.append(name, tag);
  body.append(txt);
  if (g.players) {
    const who = document.createElement('span');
    who.className = 'inline-flex shrink-0 items-center gap-1 rounded-md bg-white/10 px-[7px] py-0.5 text-xs font-semibold text-slate-400';
    who.title = 'Nombre de joueurs';
    who.append(icon('users', 'size-3.5'), g.players);
    body.append(who);
  }
  body.append(icon('chevron-right', 'size-4 text-slate-600'));
  a.append(tile, body);
  return a;
}
if ($('salonList')) $('salonList').append(menuRow('/salon.html',
  { name: 'Ouvrir un salon', icon: 'users', tile: 'bg-amber-500', tag: 'Retrouvez-vous, puis le chef choisit le jeu.' }));
if ($('tvList')) {
  $('tvList').append(
    menuRow('/tele.html', { name: 'Ouvrir l\'écran télé', icon: 'tv', tile: 'bg-slate-600', tag: 'Sur la télé ou l\'ordinateur branché dessus\u00a0: un QR code s\'affiche.' }),
    menuRow('/manette.html', { name: 'Prendre une manette', icon: 'phone', tile: 'bg-slate-600', tag: 'Sur ton téléphone\u00a0: scanne le QR code ou tape le code de la télé.' }));
  for (const g of TV_GAMES) $('tvGames').append(menuRow('/manette.html', g));
}
if ($('gameList')) for (const g of GAMES) $('gameList').append(menuRow(g.href, g)); // menu du site : une ligne par jeu
if ($('games')) { // barre d'onglets en bas de l'accueil des jeux (comme une appli)
  const here = location.pathname, row = document.createElement('div');
  $('games').className = 'fixed inset-x-0 bottom-0 z-20 border-t border-white/10 bg-slate-950/85 pb-[env(safe-area-inset-bottom)] backdrop-blur-xl';
  row.className = 'mx-auto flex max-w-2xl overflow-x-auto [scrollbar-width:none]';
  for (const g of GAMES) {
    const a = document.createElement('a'), cur = g.href === here;
    a.href = g.href;
    a.className = 'flex min-w-16 flex-1 flex-col items-center gap-0.5 pt-[7px] pb-1 text-[10px] font-medium transition active:scale-95 ' +
      (cur ? g.text : 'text-slate-500 hover:text-slate-300');
    if (cur) a.setAttribute('aria-current', 'page');
    a.append(icon(g.icon, 'size-[26px]'), g.name);
    row.append(a);
  }
  $('games').append(row);
  const cur = row.querySelector('[aria-current]'); // sur petit écran la barre défile : on centre le jeu en cours
  if (cur) row.scrollLeft = cur.offsetLeft - (row.clientWidth - cur.offsetWidth) / 2;
}
if ($('copy')) {
  const q = document.createElement('a');
  q.href = location.pathname; // sans ?pin : on quitte la partie et on revient à l'accueil du jeu
  q.title = 'Quitter la partie';
  q.className = 'inline-flex items-center gap-0.5 py-1.5 pr-2 text-[17px] transition-opacity hover:opacity-80 ' +
    (GAMES.find(g => g.href === location.pathname)?.text || 'text-slate-300');
  q.append(icon('chevron-left', 'size-5'), 'Quitter');
  const wrap = document.createElement('span');
  wrap.className = 'flex items-center gap-2';
  const copy = $('copy');
  copy.replaceWith(wrap);
  wrap.append(q, copy);
  if (location.pathname !== '/salon.html') { // en partie : tout le monde retourne au salon choisir un autre jeu, sans se ré-inviter
    const back = document.createElement('button');
    back.title = 'Retourner au salon avec tout le monde pour choisir un autre jeu';
    back.className = 'inline-flex items-center gap-1.5 rounded-full bg-white/10 px-3 py-1.5 text-sm font-semibold text-slate-200 transition hover:bg-white/15';
    back.append(icon('users', 'size-4'), 'Salon');
    back.onclick = () => send({ go: '/salon.html' });
    wrap.append(back);
  }
}
const show = (el, v) => v ? el.removeAttribute('hidden') : el.setAttribute('hidden', '');

// ---------- Style au choix : Classique, Y2K ou Liquid Glass (app.css), gardé d'une page à l'autre ----------
// La page l'applique dès son en-tête (localStorage.theme) ; ici, le bouton « Style » en haut à droite des accueils.
const THEMES = [
  ['', 'Classique', 'linear-gradient(135deg,#1e293b,#020617)'],
  ['y2k', 'Y2K', 'radial-gradient(120% 100% at 50% 0%,#fff 0%,#fff0 45%),linear-gradient(#3b86ff,#1d4fb8)'],
  ['glass', 'Liquid Glass', 'linear-gradient(135deg,#ff5fb4,#4f7bff 50%,#1fd1b5)'],
];
function setTheme(t) {
  if (t) { document.documentElement.dataset.theme = t; localStorage.theme = t; }
  else { delete document.documentElement.dataset.theme; localStorage.removeItem('theme'); }
}
const themePick = ($('home') || $('choix')) && (() => {
  const wrap = document.createElement('div'), btn = document.createElement('button'), menu = document.createElement('div');
  wrap.className = 'theme-pick';
  btn.title = 'Choisir le style'; btn.setAttribute('aria-haspopup', 'true'); btn.setAttribute('aria-expanded', 'false');
  btn.append(icon('sparkles', 'size-4'), 'Style');
  menu.hidden = true; menu.setAttribute('role', 'menu');
  const paint = () => menu.querySelectorAll('button').forEach((b, i) => show(b.lastChild, (document.documentElement.dataset.theme || '') === THEMES[i][0])); // coche : le style en cours
  for (const [id, name, sw] of THEMES) {
    const b = document.createElement('button'), s = document.createElement('span'), n = document.createElement('span');
    b.setAttribute('role', 'menuitemradio');
    s.className = 'swatch'; s.style.background = sw;
    n.className = 'flex-1'; n.textContent = name;
    b.append(s, n, icon('check', 'size-4 text-emerald-400'));
    b.onclick = () => { setTheme(id); paint(); };
    menu.append(b);
  }
  const open = v => { menu.hidden = !v; btn.setAttribute('aria-expanded', v); };
  btn.onclick = e => { e.stopPropagation(); open(menu.hidden); };
  addEventListener('click', e => { if (!wrap.contains(e.target)) open(false); });
  addEventListener('keydown', e => { if (e.key === 'Escape') open(false); });
  paint();
  wrap.append(btn, menu);
  document.body.append(wrap);
  return wrap;
})();
if ($('name')) { // pages de jeu seulement (pas le menu)
  $('name').value = localStorage.name || '';
  $('pin').value = new URLSearchParams(location.search).get('pin') || '';
}

function setErr(msg, isError = true) {
  $('err').textContent = msg;
  show($('errBox'), !!msg);
  show($('errIcon'), isError && !!msg);
  $('errBox').classList.toggle('text-red-400', isError);
  $('errBox').classList.toggle('text-slate-400', !isError);
}
if ($('err')) setErr('');

// (pas « screen » : ce nom cacherait window.screen, dont on a besoin pour l'orientation)
function showScreen(game) { if (themePick) show(themePick, !game); if ($('home')) show($('home'), !game); if ($('game')) show($('game'), game); if ($('games')) show($('games'), !game); fitGame(); }

// ---------- Ordi et téléphone : le terrain prend toute la place que laisse l'écran ----------
// TOUCH : écran tactile (téléphone, tablette). Les aides clavier (data-kbd) s'y cachent, les commandes au doigt (data-touch) s'y montrent.
const TOUCH = matchMedia('(pointer: coarse)').matches;
document.querySelectorAll('[data-kbd]').forEach(e => { e.hidden = TOUCH; });
document.querySelectorAll('[data-touch]').forEach(e => { e.hidden = !TOUCH; });

// la scène : le bloc qui contient le canvas, posé directement dans #game (pas dans Dessine, qui a sa propre mise en page)
const gameStage = $('c') && $('game') && $('c').parentElement.parentElement === $('game') ? $('c').parentElement : null;
let sideCol = null, gameOrder = null, rotateHint = null;
if (gameStage) {
  gameStage.classList.add('stage');
  const ratio = $('c').width / $('c').height;
  if (TOUCH && ratio > 1.4) { // terrain large, téléphone debout : on propose de le tourner
    rotateHint = document.createElement('p');
    rotateHint.className = 'flex items-center justify-center gap-1.5 text-xs text-slate-400';
    rotateHint.append(icon('arrow-path', 'size-4'), 'Tourne ton téléphone : le terrain sera plus grand');
    rotateHint.hidden = true;
    gameStage.before(rotateHint);
  }
  if (TOUCH && document.fullscreenEnabled && $('mute')) { // plein écran (Android, iPad) : plus de barre du navigateur
    const fs = document.createElement('button');
    fs.title = 'Plein écran'; fs.setAttribute('aria-label', 'Plein écran');
    fs.className = 'grid size-10 place-items-center rounded-full bg-white/10 text-slate-300 transition hover:bg-white/15';
    fs.append(icon('fullscreen', 'size-5'));
    fs.onclick = async () => {
      if (document.fullscreenElement) return document.exitFullscreen();
      try {
        await document.documentElement.requestFullscreen();
        if (ratio > 1.4) await screen.orientation?.lock?.('landscape');
      } catch { /* refusé ou pas pris en charge : on reste comme on est */ }
    };
    $('mute').before(fs);
  }
  addEventListener('resize', fitLater);
  const ro = new ResizeObserver(fitLater); // l'en-tête qui passe sur deux lignes, un message qui apparaît…
  for (const k of [$('game'), ...$('game').children]) ro.observe(k);
}
let fitQueued = false;
function fitLater() { if (!fitQueued) { fitQueued = true; requestAnimationFrame(() => { fitQueued = false; fitGame(); }); } }

function fitGame() {
  const g = $('game');
  if (!gameStage || g.hidden) return;
  const cv = $('c'), ratio = cv.width / cv.height, W = innerWidth, H = innerHeight;
  // écran bas et large (téléphone couché, petite fenêtre) : terrain à gauche, le reste dans une colonne à droite
  const side = H < 560 && W > H * 1.25;
  if (side !== !!sideCol) {
    if (side) {
      gameOrder = [...g.children];
      sideCol = document.createElement('div');
      sideCol.className = 'side-col';
      sideCol.append(...gameOrder.filter(k => k !== gameStage && getComputedStyle(k).position !== 'fixed'));
      g.append(sideCol);
    } else {
      g.append(...gameOrder.filter(k => k.isConnected)); // ordre d'origine
      sideCol.remove(); sideCol = null;
    }
    g.classList.toggle('side', side);
  }
  if (rotateHint) rotateHint.hidden = !(TOUCH && H > W);
  if (side) {
    gameStage.style.width = Math.min((H - 16) * ratio, W - 16 - 12 - 240) + 'px'; // 240 : largeur mini de la colonne (15rem)
    g.style.maxWidth = '';
    return;
  }
  gameStage.style.width = '';
  // hauteur prise par l'en-tête, les scores, le pied… : le terrain a le reste, et la page s'élargit d'autant (jusqu'à 1600 px)
  const cs = getComputedStyle(g), gap = parseFloat(cs.rowGap) || 0;
  let other = parseFloat(cs.paddingTop) + parseFloat(cs.paddingBottom);
  for (const k of g.children) if (k !== gameStage && !k.hidden && getComputedStyle(k).position !== 'fixed') other += k.offsetHeight + gap;
  const w = Math.round(Math.min(Math.max((H - other) * ratio, 360), 1600) + parseFloat(cs.paddingLeft) + parseFloat(cs.paddingRight));
  if (Math.abs((parseFloat(g.style.maxWidth) || 0) - w) > 4) g.style.maxWidth = w + 'px';
}

// ---------- Connexion, avec reconnexion automatique ----------
// Le serveur donne un jeton secret à l'arrivée et garde la place 20 s si la connexion lâche (réseau, écran
// verrouillé…). On revient alors tout seul avec ce jeton : même joueur, même score, même tour.
let ws = null, me = 0, token = '', link = null, lastMsg = 0, retries = 0;
// groupe : arrivé d'un autre jeu avec ses amis (?group=…), on crée ou rejoint tout seul le salon du groupe
let group = new URLSearchParams(location.search).get('group') || '';
if (group && $('create')) addEventListener('DOMContentLoaded', () => $('create').click());
const send = m => ws && ws.readyState === 1 && ws.send(JSON.stringify(m));

// connect valide pseudo et code, ouvre le WebSocket et appelle onState(état) à chaque image, onState(null) à la déconnexion.
// fixedName : pour la télé du mode télé, qui n'a pas de pseudo à saisir.
function connect(path, pin, extra, onState, fixedName) {
  const name = fixedName || $('name').value.trim();
  if (!name) return $('name').reportValidity();
  if (pin !== '' && !/^\d{4}$/.test(pin)) return setErr('Le code fait 4 chiffres');
  if (!fixedName) localStorage.name = name;
  audio(); // le navigateur n'autorise le son qu'après un clic
  setErr('Connexion…', false);
  link = { url: (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + path + '?name=' + encodeURIComponent(name) + '&pin=' + pin + extra + (group ? '&group=' + encodeURIComponent(group) : ''), onState };
  token = '';
  dial();
}

function dial() {
  const sock = ws = new WebSocket(link.url + (token ? '&resume=' + token : ''));
  lastMsg = Date.now();
  sock.onmessage = e => {
    if (sock !== ws) return;
    lastMsg = Date.now();
    const m = JSON.parse(e.data);
    if (m.go) { ws = null; sock.close(1000); location.href = m.go + '?group=' + encodeURIComponent(m.group); return; }
    if (m.you) { me = m.you; token = m.token; group = m.group || group; retries = 0; reconnecting(false); showScreen(true); setErr(''); return; }
    if (m.chat) return chatShow(m.chat);
    if (m.pin && location.search !== '?pin=' + m.pin) history.replaceState(null, '', '?pin=' + m.pin); // l'URL devient un lien d'invitation
    link.onState(m);
  };
  sock.onclose = e => {
    if (sock !== ws) return;
    // coupure : on retente avec le jeton tant que le serveur garde la place (sinon il répond 1008 avec la raison)
    if (token && e.code !== 1008 && retries < 15) {
      reconnecting(true);
      setTimeout(dial, [500, 1000, 2000][retries++] || 3000);
      return;
    }
    token = ''; reconnecting(false); chatClear(); link.onState(null); showScreen(false); setErr(e.reason || 'Déconnecté');
  };
}

// connexion figée (téléphone réveillé, wifi → 4G) : les jeux envoient en continu, donc 4 s sans rien = elle est morte.
// On n'attend pas qu'elle finisse de mourir : on en ouvre une nouvelle, le serveur lui passe la place.
function checkAlive() {
  if (!ws || ws.readyState !== 1 || !token || Date.now() - lastMsg < 4000) return;
  const old = ws;
  reconnecting(true);
  dial();
  old.close(4000, 'reprise');
}
setInterval(checkAlive, 1000);
addEventListener('visibilitychange', () => { if (!document.hidden) checkAlive(); });

// ---------- Chat de la partie : relayé en direct par le serveur, jamais enregistré ----------
let chatOpen = false, unread = 0;
const chatUI = $('game') && !('nochat' in $('game').dataset) && (() => { // pas de chat en mode télé
  const mk = (tag, cls, ...kids) => { const e = document.createElement(tag); e.className = cls; e.append(...kids); return e; };
  const badge = mk('span', 'absolute -top-1 -right-1 grid min-w-5 place-items-center rounded-full bg-red-500 px-1 text-[11px] font-bold text-white', '');
  const btn = mk('button', 'fixed right-4 bottom-4 z-40 grid size-12 place-items-center rounded-full bg-slate-800/90 text-xl shadow-2xl ring-1 ring-white/10 backdrop-blur-xl transition hover:bg-slate-700 active:scale-95', icon('chat', 'size-6'), badge);
  btn.title = 'Chat de la partie'; btn.setAttribute('aria-label', 'Chat de la partie');
  const list = mk('ul', 'flex max-h-64 min-h-16 flex-col gap-1 overflow-y-auto p-3 text-sm', '');
  const input = mk('input', 'h-10 min-w-0 flex-1 rounded-lg bg-slate-900 px-3 text-[15px] ring-1 ring-white/10 placeholder:text-slate-500 focus:ring-2 focus:ring-amber-400 focus:outline-none', '');
  input.maxLength = 200; input.placeholder = 'Écris un message…'; input.autocomplete = 'off'; input.setAttribute('aria-label', 'Message');
  // les touches tapées dans le chat ne doivent pas piloter le jeu (Z Q S D, M, B…)
  for (const ev of ['keydown', 'keyup']) input.addEventListener(ev, e => e.stopPropagation());
  const form = mk('form', 'flex gap-2 border-t border-white/10 p-2', input,
    mk('button', 'h-10 rounded-lg bg-amber-400 px-3 font-semibold text-slate-950 transition hover:bg-amber-300 active:scale-95', 'Envoyer'));
  const panel = mk('div', 'fixed right-4 bottom-20 z-40 flex w-80 max-w-[calc(100vw-2rem)] flex-col rounded-[14px] bg-slate-800/90 shadow-2xl ring-1 ring-white/10 backdrop-blur-xl',
    mk('div', 'border-b border-white/10 px-3 py-2 text-[13px] text-slate-400', 'Chat de la partie · rien n\'est enregistré'), list, form);
  panel.hidden = true;
  form.onsubmit = e => { e.preventDefault(); const t = input.value.trim(); if (t) send({ chat: t }); input.value = ''; };
  btn.onclick = () => { chatOpen = !chatOpen; panel.hidden = !chatOpen; if (chatOpen) { unread = 0; input.focus(); } paint(); };
  const paint = () => { badge.textContent = unread; badge.hidden = !unread; };
  paint();
  $('game').append(btn, panel);
  return { list, paint };
})();

function chatShow(c) {
  if (!chatUI) return;
  const li = document.createElement('li');
  if (!c.from) { li.className = 'text-slate-400 italic'; li.textContent = c.text; } // message du serveur
  else {
    const who = document.createElement('b');
    who.className = c.id === me ? 'text-amber-300' : 'text-slate-100';
    who.textContent = c.from + ' ';
    li.className = 'break-words text-slate-200';
    li.append(who, c.text);
  }
  chatUI.list.append(li);
  while (chatUI.list.children.length > 100) chatUI.list.firstChild.remove();
  chatUI.list.scrollTop = chatUI.list.scrollHeight;
  if (!chatOpen && c.id !== me) { unread++; chatUI.paint(); tone(990, 0.06, 'sine', 0.06); }
}
function chatClear() { if (chatUI) { chatUI.list.replaceChildren(); unread = 0; chatUI.paint(); } }

// ---------- Mouvements fluides (foot, billard) ----------
// Chaque état reçu est daté sur une horloge régulière (un tous les tickMs) qui suit doucement l'arrivée réelle :
// les à-coups du réseau sont absorbés. L'affichage a 2,5 états de retard pour toujours avoir de quoi interpoler.
function snapBuffer(tickMs) {
  const delay = tickMs * 2.5;
  let snaps = [];
  return {
    clear() { snaps = []; },
    push(s) {
      const now = performance.now(), last = snaps.at(-1);
      let t = last ? last.t + tickMs : now;
      t += (now - t) * 0.1; // rattrape la dérive sans saccade
      if (Math.abs(now - t) > 250) t = now; // longue coupure : on repart de zéro
      snaps.push({ t, s });
      while (snaps.length > 2 && snaps[1].t < now - delay - tickMs) snaps.shift();
    },
    // les deux états autour de l'instant affiché, et où on en est de l'un à l'autre (0 à 1)
    at(now) {
      const rt = now - delay;
      let i = 0;
      while (i < snaps.length - 2 && snaps[i + 1].t <= rt) i++;
      const a = snaps[i], b = snaps[i + 1] || a;
      return { A: a.s, B: b.s, a: b === a ? 1 : Math.min(Math.max((rt - a.t) / (b.t - a.t), 0), 1) };
    },
  };
}
// position de q, glissée depuis p ; grand saut (replacement, remise en jeu) : on y va directement
const glide = (p, q, a) => p && Math.hypot(q.x - p.x, q.y - p.y) < 60 ? { ...q, x: p.x + (q.x - p.x) * a, y: p.y + (q.y - p.y) * a } : q;

let banner = null;
function reconnecting(on) {
  if (on && !banner) {
    banner = document.createElement('div');
    banner.setAttribute('role', 'status');
    banner.className = 'fixed inset-x-0 top-3 z-50 mx-auto flex w-fit items-center gap-2 rounded-full bg-slate-800/90 px-4 py-2 text-sm font-semibold text-amber-300 shadow-2xl backdrop-blur-xl';
    banner.append(icon('arrow-path', 'size-4 animate-spin'), 'Reconnexion…');
    document.body.append(banner);
  }
  if (!on && banner) { banner.remove(); banner = null; }
}

if ($('copy')) $('copy').onclick = async () => {
  const url = location.href;
  // l'API presse-papiers n'existe qu'en https ou sur localhost
  if (!navigator.clipboard) return prompt('Lien d\'invitation à copier :', url);
  await navigator.clipboard.writeText(url);
  show($('copyLink'), false); show($('copyOk'), true);
  setTimeout(() => { show($('copyOk'), false); show($('copyLink'), true); }, 1500);
};

// ---------- Son (synthétisé avec Web Audio, aucun fichier) ----------
let actx = null, muted = localStorage.muted === '1';
function audio() {
  if (!actx) actx = new AudioContext();
  if (actx.state === 'suspended') actx.resume();
}
function setMuted(v) {
  muted = v; localStorage.muted = v ? '1' : '0';
  show($('soundOn'), !v); show($('soundOff'), v);
}
if ($('mute')) {
  setMuted(muted);
  $('mute').onclick = () => { setMuted(!muted); $('mute').blur(); };
}

function tone(freq, dur, type = 'sine', vol = 0.15, to = 0, delay = 0) {
  if (muted || !actx) return;
  const t = actx.currentTime + delay, o = actx.createOscillator(), g = actx.createGain();
  o.type = type; o.frequency.setValueAtTime(freq, t);
  if (to) o.frequency.exponentialRampToValueAtTime(to, t + dur);
  g.gain.setValueAtTime(vol, t); g.gain.exponentialRampToValueAtTime(0.001, t + dur);
  o.connect(g).connect(actx.destination); o.start(t); o.stop(t + dur);
}
function noise(dur, vol) {
  if (muted || !actx) return;
  const t = actx.currentTime, buf = actx.createBuffer(1, actx.sampleRate * dur, actx.sampleRate), d = buf.getChannelData(0);
  for (let i = 0; i < d.length; i++) d[i] = Math.random() * 2 - 1;
  const src = actx.createBufferSource(), f = actx.createBiquadFilter(), g = actx.createGain();
  src.buffer = buf; f.type = 'lowpass';
  f.frequency.setValueAtTime(3000, t); f.frequency.exponentialRampToValueAtTime(80, t + dur);
  g.gain.setValueAtTime(vol, t); g.gain.exponentialRampToValueAtTime(0.001, t + dur);
  src.connect(f).connect(g).connect(actx.destination); src.start(t);
}
