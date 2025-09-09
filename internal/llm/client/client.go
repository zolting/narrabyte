package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"narrabyte/internal/llm/tools"
	"strconv"

	appUtils "narrabyte/internal/utils"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

type OpenAIClient struct {
	ChatModel openai.ChatModel
	//GitToolsService tools.GitToolsService
	Key string
}

func NewOpenAIClient(ctx context.Context, key string) (*OpenAIClient, error) {
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: key,
		Model:  "gpt-5-mini",
	})

	if err != nil {
		log.Printf("Error creating OpenAI client: %v", err)
		return nil, err
	}

	return &OpenAIClient{*model, key}, err
}

// SetListDirectoryBaseRoot binds the list-directory tools to a specific base directory.
// Example: SetListDirectoryBaseRoot("/path/to/project") then tool input "frontend"
// resolves to "/path/to/project/frontend".
func (o *OpenAIClient) SetListDirectoryBaseRoot(root string) {
	tools.SetListDirectoryBaseRoot(root)
}

//La fonction essaie d'appeler un modèle de chat avec un outil d'addition, d'exécuter le flux "LLM -> Tools" puis de retourner le dernier message produit. Elle prépare les messages, attache l'outil au modèle, construit une chaîne composée d'un nœud LLM puis d'un nœud Tools, compile et invoque l'agent.
//
//Étapes détaillées :
//
//
//Infère un outil nommé add_tool via utils.InferTool.
//Construit la liste messages (System + User) avec la question d'addition.
//Appelle addTool.Info(ctx) mais ignore l'erreur (ligne actuelle).
//Lie les métadonnées de l'outil au modèle via o.ChatModel.BindForcedTools.
//Crée un toolsNode contenant l'outil obtenu.
//Crée une chain et y ajoute un nœud ChatModel (llm-plan) puis le toolsNode.
//Compile la chaîne en agent.
//Invoque agent.Invoke(ctx, messages) pour exécuter la composition.
//Vérifie que outMsg n'est pas nil, puis tente de renvoyer le contenu du dernier message avec outMsg[len(outMsg)-1].Content.
//Points problématiques / risques (à corriger) :
//
//
//info, _ := addTool.Info(ctx) ignore l'erreur : si info est nil, la liaison et l'exécution échoueront.
//outMsg est de type *[]schema.Message (pointeur vers slice) — on ne peut pas faire outMsg[len(outMsg)-1]. Il faut d'abord vérifier outMsg != nil, puis faire msgs := *outMsg et indexer msgs[len(msgs)-1].
//Il manque des contrôles d'erreur supplémentaires (vérifier la compilation, la réussite de la liaison des outils, et la longueur du slice avant d'indexer).

// Merci chat pour le resume :)

// Etant donne que c'est une demo simple, le context est Background() mais il faudrait peut-etre mettre un timeout
func (o *OpenAIClient) InvokeAdditionDemo(ctx context.Context, a, b int) (string, error) {
	addTool, err := utils.InferTool("add_tool", "adds two integers and gives the result", tools.Add)
	if err != nil {
		log.Printf("Error inferring tool: %v", err)
		return "", err
	}

	messages := []*schema.Message{
		schema.SystemMessage("You are a helpful assistant that can perform addition using the provided tool."),
		schema.UserMessage("What is the sum of" + strconv.Itoa(a) + "and" + strconv.Itoa(b) + "?"),
	}

	info, err := addTool.Info(ctx)
	if err != nil {
		log.Printf("Error getting tool info: %v", err)
		return "", err
	}

	if err := o.ChatModel.BindForcedTools([]*schema.ToolInfo{info}); err != nil {
		log.Printf("Error binding tools: %v", err)
		return "", err
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{addTool},
	})
	if err != nil {
		log.Printf("Error creating tools node: %v", err)
		return "", err
	}

	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.AppendChatModel(&o.ChatModel, compose.WithNodeName("llm-plan"))
	chain.AppendToolsNode(toolsNode, compose.WithNodeName("tools"))

	agent, err := chain.Compile(ctx)
	if err != nil {
		log.Printf("Error compiling chain: %v", err)
		return "", err
	}

	outMsg, err := agent.Invoke(ctx, messages)
	if err != nil {
		log.Printf("Error invoking agent: %v", err)
		return "", err
	}
	if outMsg == nil {
		return "", fmt.Errorf("agent returned no message")
	}

	return outMsg[len(outMsg)-1].Content, nil
}

func (o *OpenAIClient) InvokeListDirectoryDemo(ctx context.Context, repoPath string, opts tools.TreeOptions) (string, error) {
	// Build the list-directory tool
	listDirectoryTool, err := utils.InferTool("list_directory_tool", "lists the contents of a directory", tools.ListDirectoryJSON)
	if err != nil {
		log.Printf("Error inferring tool: %v", err)
		return "", err
	}

	// Configure the tool's base root (project root)
	root, err := appUtils.FindProjectRoot()
	if err != nil {
		log.Printf("Error finding project root: %v", err)
		return "", err
	}
	tools.SetListDirectoryBaseRoot(root)

	// Create an initial preview of the repo tree for context
	preview, err := tools.ListDirectoryJSON(ctx, &tools.ListDirectoryInput{
		DirectoryRelativePath: repoPath,
	})
	if err != nil {
		log.Printf("Error listing tree JSON: %v", err)
		return "", err
	}
	previewBytes, err := json.MarshalIndent(preview, "", "  ")
	if err != nil {
		log.Printf("Error marshalling tree JSON: %v", err)
		return "", err
	}

	// Messages to drive the agent
	messages := []*schema.Message{
		schema.UserMessage(
			"Here is an initial preview of the project's directory (depth 2):\n\n" +
				string(previewBytes) +
				"\n\nWhat files in the project take care of synchronizing the app settings, both backend and frontend?"),
	}

	// Build a ReAct agent with the tool-callable model and tools config
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &o.ChatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{listDirectoryTool},
		},
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			// Add a concise system persona before user / history
			res := make([]*schema.Message, 0, len(input)+1)
			res = append(res, schema.SystemMessage(
				"You are a helpful codebase assistant. "+
					"You can call list_directory_tool to inspect the repository when needed."))
			res = append(res, input...)
			return res
		},
		MaxStep: 20,
	})
	if err != nil {
		log.Printf("Error creating react agent: %v", err)
		return "", err
	}

	// Run the agent to completion and return its final message
	out, err := agent.Generate(ctx, messages)
	if err != nil {
		log.Printf("Error running react agent: %v", err)
		return "", err
	}
	if out == nil {
		return "", fmt.Errorf("agent returned no message")
	}

	println("OUT MESSAGE CONTENT : \n\n", out.Content)
	return out.Content, nil
}
