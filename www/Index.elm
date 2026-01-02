port module Index exposing (main)

import Browser
import Browser.Navigation as Nav
import Html exposing (..)
import Html.Attributes exposing (..)
import Html.Events as Events
import Http
import Json.Decode as Decode
import Json.Encode as Encode


main : Program () Model Msg
main =
    Browser.document
        { init = init
        , view = view
        , update = update
        , subscriptions = subscriptions
        }


{-| Ask js to generate a new public/private key pair with WebAuthn
-}
port createUserCredential : Encode.Value -> Cmd a


{-| Recieve the new credential so we can send it to the server and finish registration
-}
port recieveNewUserCredential : (Decode.Value -> a) -> Sub a


{-| Get any existing credentials to login with
-}
port getUserCredential : Encode.Value -> Cmd a


{-| Recieve the existing credential chosen by the user to login with
-}
port recieveUserCredential : (Decode.Value -> a) -> Sub a


type alias Model =
    { username : String
    , uiDisabled : Bool
    , message : String
    }


type Msg
    = BeginLogin
    | BeginLoginResponse (Result Http.Error Decode.Value)
    | FinishLogin Decode.Value
    | FinishLoginResponse (Result Http.Error ())
    | BeginRegister
    | BeginRegisterResponse (Result Http.Error Decode.Value)
    | FinishRegister Decode.Value
    | FinishRegisterResponse (Result Http.Error ())
    | UserNameInput String


init : () -> ( Model, Cmd Msg )
init () =
    ( { username = "", uiDisabled = False, message = "" }
    , Cmd.none
    )


subscriptions : Model -> Sub Msg
subscriptions _ =
    Sub.batch
        [ recieveNewUserCredential FinishRegister
        , recieveUserCredential FinishLogin
        ]


update : Msg -> Model -> ( Model, Cmd Msg )
update msg model =
    case msg of
        BeginLogin ->
            ( { model | message = "Logging in...", uiDisabled = True }
            , Http.post
                { url = "/api/login/begin"
                , body = Http.emptyBody
                , expect = Http.expectJson BeginLoginResponse Decode.value
                }
            )

        BeginLoginResponse result ->
            case result of
                Ok data ->
                    ( { model | message = "Waiting for authenticaor..." }
                    , getUserCredential data
                    )

                Err error ->
                    ( { model | message = httpErrorToString error, uiDisabled = False }
                    , Cmd.none
                    )

        FinishLogin value ->
            ( { model | message = "Sending credential..." }
            , Http.post
                { url = "/api/login/finish"
                , body = Http.jsonBody value
                , expect = Http.expectWhatever FinishLoginResponse
                }
            )

        FinishLoginResponse result ->
            case result of
                Ok () ->
                    ( { model | message = "omg it worked wow", uiDisabled = False }
                    , Nav.load "/server"
                    )

                Err error ->
                    ( { model
                        | message = "There was a problem logging you in: " ++ httpErrorToString error
                        , uiDisabled = False
                      }
                    , Cmd.none
                    )

        UserNameInput username ->
            ( { model | username = username }, Cmd.none )

        BeginRegister ->
            ( { model | uiDisabled = True, message = "Starting registration..." }
            , Http.post
                { url = "/api/register/begin"
                , body = Http.emptyBody
                , expect = Http.expectJson BeginRegisterResponse Decode.value
                }
            )

        BeginRegisterResponse result ->
            case result of
                Ok data ->
                    ( { model | message = "Waiting for authenticator..." }
                    , createUserCredential data
                    )

                Err error ->
                    ( { model | uiDisabled = False, message = "Registration failed: " ++ httpErrorToString error }
                    , Cmd.none
                    )

        FinishRegister value ->
            ( { model | message = "Completing registration..." }
            , Http.post
                { url = "/api/register/finish"
                , body = Http.jsonBody value
                , expect = Http.expectWhatever FinishRegisterResponse
                }
            )

        FinishRegisterResponse result ->
            case result of
                Ok () ->
                    ( { model | uiDisabled = False, message = "Registration successful!" }
                    , Nav.load "/server"
                    )

                Err error ->
                    ( { model | uiDisabled = False, message = "Registration failed: " ++ httpErrorToString error }
                    , Cmd.none
                    )


view : Model -> Browser.Document Msg
view model =
    { title = "Secret Santa", body = viewBody model }


viewBody : Model -> List (Html Msg)
viewBody model =
    [ div
        [ class "jumbotron" ]
        [ h1 [] [ text "🎅 Secret Santa 🎄" ] ]
    , div []
        [ button
            [ Events.onClick BeginLogin
            , disabled model.uiDisabled
            ]
            [ text "Login" ]
        ]
    , div []
        [ button
            [ Events.onClick BeginRegister
            , disabled model.uiDisabled
            ]
            [ text "Register" ]
        ]
    , if model.message /= "" then
        div [] [ text model.message ]

      else
        text ""
    ]


httpErrorToString : Http.Error -> String
httpErrorToString error =
    case error of
        Http.BadUrl url ->
            "Bad URL: " ++ url

        Http.Timeout ->
            "Request timeout"

        Http.NetworkError ->
            "Network error"

        Http.BadStatus status ->
            "Bad status: " ++ String.fromInt status

        Http.BadBody body ->
            "Bad body: " ++ body
